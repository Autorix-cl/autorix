"use client";

import { useState } from "react";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { Plus, Copy, AlertTriangle, Check } from "lucide-react";
import { CreateKeySchema, CreateKeyInput } from "@/lib/schemas/vulcan";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";

const AVAILABLE_SCOPES = [
  { id: "read:users", label: "Read Users" },
  { id: "write:users", label: "Write Users" },
  { id: "read:billing", label: "Read Billing" },
  { id: "read:audit", label: "Read Audit Logs" },
];

export function KeyBuilderSheet() {
  const [open, setOpen] = useState(false);
  const [showSecretModal, setShowSecretModal] = useState(false);
  const [newSecret, setNewSecret] = useState<string>("");
  const [copied, setCopied] = useState(false);
  
  const [name, setName] = useState("");
  const [expiresIn, setExpiresIn] = useState<string>("30d");
  const [scopes, setScopes] = useState<string[]>([]);
  const [errors, setErrors] = useState<Record<string, string>>({});

  const queryClient = useQueryClient();

  const resetForm = () => {
    setName("");
    setExpiresIn("30d");
    setScopes([]);
    setErrors({});
  };

  const createMutation = useMutation({
    mutationFn: async (data: CreateKeyInput) => {
      const res = await fetch("/api/vulcan/keys", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || "Failed to create key");
      }
      return res.json();
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["vulcan-keys"] });
      setNewSecret(data.secret);
      setOpen(false);
      setShowSecretModal(true);
      resetForm();
    },
    onError: (err: Error) => {
      toast.error(err.message);
    }
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    const payload = {
      name,
      expires_in: expiresIn,
      scopes,
    };
    
    const result = CreateKeySchema.safeParse(payload);
    
    if (!result.success) {
      const fieldErrors: Record<string, string> = {};
      for (const issue of result.error.issues) {
        if (issue.path[0]) {
          fieldErrors[issue.path[0].toString()] = issue.message;
        }
      }
      setErrors(fieldErrors);
      return;
    }
    
    setErrors({});
    createMutation.mutate(result.data as CreateKeyInput);
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(newSecret);
    setCopied(true);
    toast.success("Secret copied to clipboard");
    setTimeout(() => setCopied(false), 2000);
  };

  const handleCloseSecretModal = () => {
    setShowSecretModal(false);
    setNewSecret("");
  };

  return (
    <>
      <Sheet open={open} onOpenChange={(o) => {
        setOpen(o);
        if (!o) resetForm();
      }}>
        <SheetTrigger asChild>
          <Button>
            <Plus className="w-4 h-4 mr-2" />
            Create API Key
          </Button>
        </SheetTrigger>
        <SheetContent className="sm:max-w-md overflow-y-auto">
          <SheetHeader>
            <SheetTitle>Create new Root Key</SheetTitle>
            <SheetDescription>
              Generate a new Root Key to interact with the Autorix engines programmatically.
            </SheetDescription>
          </SheetHeader>
          
          <form onSubmit={handleSubmit} className="space-y-6 mt-6">
            <div className="space-y-2">
              <Label htmlFor="name">Key Name</Label>
              <Input 
                id="name" 
                placeholder="e.g. Production Backend" 
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
              {errors.name && <p className="text-sm text-destructive">{errors.name}</p>}
            </div>
            
            <div className="space-y-2">
              <Label htmlFor="expires">Expiration</Label>
              <Select value={expiresIn} onValueChange={setExpiresIn}>
                <SelectTrigger id="expires">
                  <SelectValue placeholder="Select expiration" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="7d">7 Days</SelectItem>
                  <SelectItem value="30d">30 Days</SelectItem>
                  <SelectItem value="90d">90 Days</SelectItem>
                  <SelectItem value="never">Never (Not recommended)</SelectItem>
                </SelectContent>
              </Select>
              {errors.expires_in && <p className="text-sm text-destructive">{errors.expires_in}</p>}
            </div>

            <div className="space-y-3">
              <Label>Base Scopes</Label>
              <div className="space-y-2 border rounded-md p-4 bg-muted/30">
                {AVAILABLE_SCOPES.map((scope) => (
                  <div key={scope.id} className="flex items-center space-x-2">
                    <Checkbox 
                      id={`scope-${scope.id}`} 
                      checked={scopes.includes(scope.id)}
                      onCheckedChange={(checked) => {
                        if (checked) {
                          setScopes([...scopes, scope.id]);
                        } else {
                          setScopes(scopes.filter(s => s !== scope.id));
                        }
                      }}
                    />
                    <label 
                      htmlFor={`scope-${scope.id}`} 
                      className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                    >
                      {scope.label} <span className="text-muted-foreground ml-1 font-mono text-xs">({scope.id})</span>
                    </label>
                  </div>
                ))}
              </div>
              {errors.scopes && <p className="text-sm text-destructive">{errors.scopes}</p>}
            </div>
            
            <div className="pt-4 flex justify-end">
              <Button type="submit" disabled={createMutation.isPending}>
                {createMutation.isPending ? "Generating..." : "Generate Key"}
              </Button>
            </div>
          </form>
        </SheetContent>
      </Sheet>

      <Dialog open={showSecretModal} onOpenChange={handleCloseSecretModal}>
        <DialogContent className="sm:max-w-md" onInteractOutside={(e) => e.preventDefault()} onEscapeKeyDown={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle className="flex items-center text-amber-500">
              <AlertTriangle className="w-5 h-5 mr-2" />
              Save your secret key
            </DialogTitle>
            <DialogDescription>
              Please copy this API key and store it securely. For security reasons, <strong className="text-foreground">we will never show it to you again</strong> after you close this dialog.
            </DialogDescription>
          </DialogHeader>
          
          <div className="flex items-center space-x-2 mt-4">
            <div className="grid flex-1 gap-2">
              <Label htmlFor="secret" className="sr-only">
                Secret
              </Label>
              <Input
                id="secret"
                defaultValue={newSecret}
                readOnly
                className="font-mono text-sm bg-muted"
              />
            </div>
            <Button type="button" size="icon" className="px-3" onClick={handleCopy}>
              <span className="sr-only">Copy</span>
              {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
            </Button>
          </div>
          
          <DialogFooter className="sm:justify-end mt-4">
            <Button type="button" variant="default" onClick={handleCloseSecretModal}>
              I have saved the key
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
