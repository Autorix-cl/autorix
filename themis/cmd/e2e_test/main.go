package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/autorix/themis/api/autorix/themis/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewThemisServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tenantID := "tenant_erp_acme"
	fmt.Println("=== 🚀 Running E2E Test Suite for Themis (Motor #7) ===")

	// 1. Create Policy 1: Purchase Order Approval
	p1, err := client.CreatePolicy(ctx, &pb.CreatePolicyRequest{
		TenantId:    tenantID,
		Name:        "purchase_order_approval",
		Description: "Rules for PO approval limit based on role and budget",
		Expression:  "payload.amount <= 10000.0 || (payload.role == 'cfo' && payload.amount <= 50000.0)",
		Priority:    1,
		Enabled:     true,
		Labels:      map[string]string{"module": "procurement", "tier": "financial"},
	})
	if err != nil {
		log.Fatalf("❌ Failed CreatePolicy 1: %v", err)
	}
	fmt.Printf("✅ Policy Created: %s (ID: %s)\n", p1.Policy.Name, p1.Policy.Id)

	// 2. Create Policy 2: Credit Limit Check
	p2, err := client.CreatePolicy(ctx, &pb.CreatePolicyRequest{
		TenantId:    tenantID,
		Name:        "credit_limit_check",
		Description: "Verify customer credit line is active and not exceeded",
		Expression:  "payload.customer_status == 'active' && (payload.credit_used + payload.amount <= payload.credit_limit)",
		Priority:    2,
		Enabled:     true,
		Labels:      map[string]string{"module": "sales", "tier": "risk"},
	})
	if err != nil {
		log.Fatalf("❌ Failed CreatePolicy 2: %v", err)
	}
	fmt.Printf("✅ Policy Created: %s (ID: %s)\n", p2.Policy.Name, p2.Policy.Id)

	// 3. List Policies
	listRes, err := client.ListPolicies(ctx, &pb.ListPoliciesRequest{TenantId: tenantID})
	if err != nil {
		log.Fatalf("❌ Failed ListPolicies: %v", err)
	}
	fmt.Printf("✅ Listed %d policies for tenant %s\n", len(listRes.Policies), tenantID)

	// 4. Test Evaluation Scenario A: Standard Manager PO ($8,000) -> Should PASS
	payloadA, _ := structpb.NewStruct(map[string]interface{}{
		"amount": 8000.0,
		"role":   "manager",
	})
	evalA, err := client.EvaluatePolicy(ctx, &pb.EvaluatePolicyRequest{
		TenantId: tenantID,
		PolicyId: p1.Policy.Id,
		Payload:  payloadA,
	})
	if err != nil || !evalA.AllPassed {
		log.Fatalf("❌ Evaluation A failed: %v, passed: %v", err, evalA.AllPassed)
	}
	fmt.Printf("✅ Scenario A (PO $8K Manager): ALL PASSED = %v (results=%d)\n", evalA.AllPassed, len(evalA.Results))

	// 5. Test Evaluation Scenario B: Standard Manager PO ($25,000) -> Should FAIL
	payloadB, _ := structpb.NewStruct(map[string]interface{}{
		"amount": 25000.0,
		"role":   "manager",
	})
	evalB, err := client.EvaluatePolicy(ctx, &pb.EvaluatePolicyRequest{
		TenantId: tenantID,
		PolicyId: p1.Policy.Id,
		Payload:  payloadB,
	})
	if err != nil {
		log.Fatalf("❌ Evaluation B RPC error: %v", err)
	}
	if evalB.AllPassed {
		log.Fatalf("❌ Evaluation B was supposed to FAIL but passed")
	}
	fmt.Printf("✅ Scenario B (PO $25K Manager - Over limit): Correctly REJECTED (passed=%v)\n", evalB.AllPassed)

	// 6. Test Evaluation Scenario C: CFO PO ($25,000) -> Should PASS
	payloadC, _ := structpb.NewStruct(map[string]interface{}{
		"amount": 25000.0,
		"role":   "cfo",
	})
	evalC, err := client.EvaluatePolicy(ctx, &pb.EvaluatePolicyRequest{
		TenantId: tenantID,
		PolicyId: p1.Policy.Id,
		Payload:  payloadC,
	})
	if err != nil || !evalC.AllPassed {
		log.Fatalf("❌ Evaluation C failed: %v", err)
	}
	fmt.Printf("✅ Scenario C (PO $25K CFO Override): Approved as expected (passed=%v)\n", evalC.AllPassed)

	// 7. Batch Evaluation Scenario
	payloadSales1, _ := structpb.NewStruct(map[string]interface{}{
		"customer_status": "active",
		"credit_used":     3000.0,
		"amount":          1500.0,
		"credit_limit":    10000.0,
	})
	payloadSales2, _ := structpb.NewStruct(map[string]interface{}{
		"customer_status": "suspended",
		"credit_used":     0.0,
		"amount":          100.0,
		"credit_limit":    5000.0,
	})
	batchRes, err := client.BatchEvaluate(ctx, &pb.BatchEvaluateRequest{
		TenantId:    tenantID,
		LabelFilter: map[string]string{"module": "sales"},
		Payloads:    []*structpb.Struct{payloadSales1, payloadSales2},
	})
	if err != nil {
		log.Fatalf("❌ BatchEvaluate RPC failed: %v", err)
	}
	fmt.Printf("✅ Batch Evaluation: processed %d payloads. Item 1 passed=%v, Item 2 passed=%v\n",
		len(batchRes.Evaluations),
		batchRes.Evaluations[0].AllPassed,
		batchRes.Evaluations[1].AllPassed)

	// 8. Clean up / Delete Policy
	_, err = client.DeletePolicy(ctx, &pb.DeletePolicyRequest{TenantId: tenantID, PolicyId: p1.Policy.Id})
	if err != nil {
		log.Fatalf("❌ DeletePolicy failed: %v", err)
	}
	_, err = client.DeletePolicy(ctx, &pb.DeletePolicyRequest{TenantId: tenantID, PolicyId: p2.Policy.Id})
	if err != nil {
		log.Fatalf("❌ DeletePolicy failed: %v", err)
	}
	fmt.Println("✅ Policies cleaned up successfully.")

	fmt.Println("\n🎉 === ALL END-TO-END USE CASE TESTS PASSED PERFECTLY ===")
}
