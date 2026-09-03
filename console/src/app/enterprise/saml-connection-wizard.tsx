"use client";

import * as React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Building2,
  ChevronRight,
  ChevronLeft,
  Zap,
  CheckCircle2,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";

interface SAMLConnectionWizardProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

const TEMPLATES: { [key: string]: { name: string; mapping: { [key: string]: string } } } = {
  okta: {
    name: "Okta",
    mapping: {
      "user.email": "email",
      "user.firstName": "first_name",
      "user.lastName": "last_name",
      "user.groups": "roles",
    },
  },
  azure: {
    name: "Microsoft Entra / Azure AD",
    mapping: {
      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress": "email",
      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname": "first_name",
      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname": "last_name",
    },
  },
  google: {
    name: "Google Workspace",
    mapping: {
      "email": "email",
      "first_name": "first_name",
      "last_name": "last_name",
    },
  },
};

export function SAMLConnectionWizard({
  isOpen,
  onOpenChange,
  onSuccess,
}: SAMLConnectionWizardProps) {
  const [step, setStep] = React.useState<1 | 2 | 3>(1);

  // Step 1: IdP details
  const [providerId, setProviderId] = React.useState("");
  const [displayName, setDisplayName] = React.useState("");
  const [idpEntityId, setIdpEntityId] = React.useState("");
  const [idpSsoUrl, setIdpSsoUrl] = React.useState("");
  const [certificatePem, setCertificatePem] = React.useState("");

  // Step 2: Attribute mapping
  const [template, setTemplate] = React.useState("okta");
  const [attributeMapping, setAttributeMapping] = React.useState<{ [key: string]: string }>(
    TEMPLATES.okta.mapping
  );

  // Step 3: Test status
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [testSuccess, setTestSuccess] = React.useState(false);

  React.useEffect(() => {
    if (isOpen) {
      setStep(1);
      setTestSuccess(false);
    }
  }, [isOpen]);

  const selectTemplate = (key: string) => {
    setTemplate(key);
    if (TEMPLATES[key]) {
      setAttributeMapping(TEMPLATES[key].mapping);
    }
  };

  const handleCreate = async () => {
    setIsSubmitting(true);
    try {
      const payload = {
        id: providerId.trim(),
        name: displayName.trim() || providerId.trim(),
        idpEntityId: idpEntityId.trim() || `https://sts.example.com/${providerId.trim()}`,
        ssoUrl: idpSsoUrl.trim(),
        certificatePem: certificatePem.trim() || undefined,
        attribute_mapping: attributeMapping,
      };

      const res = await fetch("/api/enterprise/saml", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || "Failed to register connection");
      }

      setTestSuccess(true);
      toast.success("SAML Provider registered and connection verified!");
      onSuccess?.();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <Building2 className="w-5 h-5 text-primary" />
            <DialogTitle>SAML Connection Setup Wizard</DialogTitle>
          </div>
          <DialogDescription className="text-xs">
            Step {step} of 3 · Configure federated enterprise authentication.
          </DialogDescription>
        </DialogHeader>

        {/* Step Indicator */}
        <div className="flex items-center justify-between text-xs py-2 border-b">
          <span className={`font-semibold ${step === 1 ? "text-primary" : "text-muted-foreground"}`}>
            1. Identity Provider
          </span>
          <ChevronRight className="w-3.5 h-3.5 text-muted-foreground" />
          <span className={`font-semibold ${step === 2 ? "text-primary" : "text-muted-foreground"}`}>
            2. Attribute Mapping
          </span>
          <ChevronRight className="w-3.5 h-3.5 text-muted-foreground" />
          <span className={`font-semibold ${step === 3 ? "text-primary" : "text-muted-foreground"}`}>
            3. Review & Connect
          </span>
        </div>

        {/* Step 1: IdP Config */}
        {step === 1 && (
          <div className="space-y-3 pt-2 text-xs">
            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-1">
                <Label className="text-[11px]">Provider Slug</Label>
                <Input
                  placeholder="e.g. okta-corporate"
                  value={providerId}
                  onChange={(e) => setProviderId(e.target.value)}
                  className="h-8 text-xs font-mono"
                  required
                />
              </div>
              <div className="space-y-1">
                <Label className="text-[11px]">Display Name</Label>
                <Input
                  placeholder="e.g. Okta Corporate"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  className="h-8 text-xs"
                />
              </div>
            </div>

            <div className="space-y-1">
              <Label className="text-[11px]">IdP SSO Service URL (HTTP-POST)</Label>
              <Input
                placeholder="https://company.okta.com/app/autorix/sso/saml"
                value={idpSsoUrl}
                onChange={(e) => setIdpSsoUrl(e.target.value)}
                className="h-8 text-xs font-mono"
                required
              />
            </div>

            <div className="space-y-1">
              <Label className="text-[11px]">IdP Entity ID (Issuer URI)</Label>
              <Input
                placeholder="https://www.okta.com/exk123456"
                value={idpEntityId}
                onChange={(e) => setIdpEntityId(e.target.value)}
                className="h-8 text-xs font-mono"
              />
            </div>

            <div className="space-y-1">
              <Label className="text-[11px]">X.509 Certificate (PEM)</Label>
              <textarea
                rows={3}
                placeholder="-----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----"
                value={certificatePem}
                onChange={(e) => setCertificatePem(e.target.value)}
                className="w-full text-xs font-mono p-2 rounded border bg-muted/20"
              />
            </div>

            <DialogFooter className="pt-2">
              <Button
                size="sm"
                onClick={() => setStep(2)}
                disabled={!providerId.trim() || !idpSsoUrl.trim()}
                className="gap-1 text-xs"
              >
                Next <ChevronRight className="w-3.5 h-3.5" />
              </Button>
            </DialogFooter>
          </div>
        )}

        {/* Step 2: Attribute Mapping */}
        {step === 2 && (
          <div className="space-y-3 pt-2 text-xs">
            <div className="space-y-1.5">
              <Label className="text-[11px]">Choose IdP Preset</Label>
              <div className="flex gap-1.5">
                {Object.entries(TEMPLATES).map(([k, v]) => (
                  <Button
                    key={k}
                    type="button"
                    variant={template === k ? "default" : "outline"}
                    size="sm"
                    className="h-7 text-xs"
                    onClick={() => selectTemplate(k)}
                  >
                    {v.name}
                  </Button>
                ))}
              </div>
            </div>

            <div className="p-3 rounded-lg border bg-card/60 space-y-2">
              <span className="font-semibold text-foreground text-[11px]">
                Active Assertion Mappings ({Object.keys(attributeMapping).length})
              </span>
              <div className="space-y-1 max-h-40 overflow-y-auto pr-1">
                {Object.entries(attributeMapping).map(([k, v]) => (
                  <div key={k} className="flex items-center justify-between text-[11px] p-1.5 rounded border bg-background font-mono">
                    <span className="truncate max-w-[200px] text-muted-foreground">{k}</span>
                    <span>➔</span>
                    <span className="text-primary font-semibold">{v}</span>
                  </div>
                ))}
              </div>
            </div>

            <DialogFooter className="pt-2 flex justify-between">
              <Button size="sm" variant="ghost" onClick={() => setStep(1)} className="gap-1 text-xs">
                <ChevronLeft className="w-3.5 h-3.5" /> Back
              </Button>
              <Button size="sm" onClick={() => setStep(3)} className="gap-1 text-xs">
                Next <ChevronRight className="w-3.5 h-3.5" />
              </Button>
            </DialogFooter>
          </div>
        )}

        {/* Step 3: Review & Connect */}
        {step === 3 && (
          <div className="space-y-4 pt-2 text-xs">
            <div className="p-3 rounded-lg border bg-card/60 space-y-2">
              <div className="font-semibold text-foreground text-xs">Summary</div>
              <div className="grid grid-cols-2 gap-2 text-[11px]">
                <div>
                  <span className="text-muted-foreground">ID: </span>
                  <span className="font-mono text-foreground">{providerId}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Name: </span>
                  <span className="text-foreground">{displayName || providerId}</span>
                </div>
                <div className="col-span-2 truncate">
                  <span className="text-muted-foreground">SSO URL: </span>
                  <span className="font-mono text-foreground">{idpSsoUrl}</span>
                </div>
              </div>
            </div>

            {testSuccess ? (
              <div className="p-3 rounded-lg border bg-emerald-500/10 border-emerald-500/30 text-emerald-500 flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 shrink-0" />
                <span>Connection established successfully!</span>
              </div>
            ) : null}

            <DialogFooter className="pt-2 flex justify-between">
              {!testSuccess && (
                <Button size="sm" variant="ghost" onClick={() => setStep(2)} className="gap-1 text-xs">
                  <ChevronLeft className="w-3.5 h-3.5" /> Back
                </Button>
              )}
              {testSuccess ? (
                <Button size="sm" onClick={() => onOpenChange(false)} className="w-full text-xs">
                  Done
                </Button>
              ) : (
                <Button size="sm" onClick={handleCreate} disabled={isSubmitting} className="gap-1.5 text-xs">
                  {isSubmitting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Zap className="w-3.5 h-3.5" />}
                  Register & Verify Connection
                </Button>
              )}
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
