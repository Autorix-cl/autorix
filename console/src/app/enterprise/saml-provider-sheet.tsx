"use client";

import * as React from "react";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  ShieldCheck,
  Building2,
  Trash2,
  Save,
  Plus,
  AlertTriangle,
  Calendar,
  CheckCircle2,
  XCircle,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Can, usePermission } from "@/lib/auth/use-permission";
import { getCsrfHeaders } from "@/lib/api/client";
import type { SAMLProvider, CertificateInfo } from "@/lib/api/schemas/hermes";

interface SAMLProviderSheetProps {
  provider: SAMLProvider | null;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function SAMLProviderSheet({ provider, isOpen, onOpenChange, onSuccess }: SAMLProviderSheetProps) {
  const { can } = usePermission();
  const canWrite = can("enterprise:write");
  const [displayName, setDisplayName] = React.useState("");
  const [ssoUrl, setSsoUrl] = React.useState("");
  const [enabled, setEnabled] = React.useState(true);
  const [attributeMapping, setAttributeMapping] = React.useState<{ [key: string]: string }>({});
  const [newKey, setNewKey] = React.useState("");
  const [newValue, setNewValue] = React.useState("");
  const [isSaving, setIsSaving] = React.useState(false);
  const [isDeleting, setIsDeleting] = React.useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = React.useState(false);

  React.useEffect(() => {
    if (provider) {
      setDisplayName(provider.display_name || provider.id);
      setSsoUrl(provider.idp_sso_url || "");
      setEnabled(provider.enabled);
      setAttributeMapping(provider.attribute_mapping || {});
    }
  }, [provider]);

  if (!provider) return null;

  const handleToggleEnabled = async (nextState: boolean) => {
    try {
      const endpoint = nextState ? "enable" : "disable";
      const res = await fetch(`/api/enterprise/saml/${encodeURIComponent(provider.id)}/${endpoint}`, {
        method: "POST",
        headers: getCsrfHeaders(),
      });
      if (!res.ok) throw new Error("Failed to toggle provider status");
      setEnabled(nextState);
      toast.success(`Provider ${nextState ? "enabled" : "disabled"}`);
      onSuccess?.();
    } catch {
      toast.error("Failed to update status");
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    try {
      const res = await fetch(`/api/enterprise/saml/${encodeURIComponent(provider.id)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", ...getCsrfHeaders() },
        body: JSON.stringify({
          display_name: displayName,
          idp_sso_url: ssoUrl,
          attribute_mapping: attributeMapping,
        }),
      });
      if (!res.ok) throw new Error("Failed to save changes");
      toast.success("SAML Provider updated");
      onSuccess?.();
      onOpenChange(false);
    } catch {
      toast.error("Failed to save provider settings");
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    setIsDeleting(true);
    try {
      const res = await fetch(`/api/enterprise/saml/${encodeURIComponent(provider.id)}`, {
        method: "DELETE",
        headers: getCsrfHeaders(),
      });
      if (!res.ok) throw new Error("Failed to delete provider");
      toast.success("Provider deleted");
      setConfirmDeleteOpen(false);
      onSuccess?.();
      onOpenChange(false);
    } catch {
      toast.error("Failed to delete provider");
    } finally {
      setIsDeleting(false);
    }
  };

  const addMapping = () => {
    if (!newKey.trim() || !newValue.trim()) return;
    setAttributeMapping({ ...attributeMapping, [newKey.trim()]: newValue.trim() });
    setNewKey("");
    setNewValue("");
  };

  const removeMapping = (keyToRemove: string) => {
    const next = { ...attributeMapping };
    delete next[keyToRemove];
    setAttributeMapping(next);
  };

  const certs: CertificateInfo[] = provider.certificates || [];

  return (
    <>
      <Sheet open={isOpen} onOpenChange={onOpenChange}>
        <SheetContent className="sm:max-w-lg flex flex-col">
          <SheetHeader>
            <div className="flex items-center justify-between">
              <SheetTitle className="flex items-center gap-2 text-sm">
                <Building2 className="w-4 h-4 text-primary" />
                {provider.display_name || provider.id}
              </SheetTitle>
              <div className="flex items-center gap-2 pr-6">
                <Switch
                  checked={enabled}
                  disabled={!canWrite}
                  aria-label={`Toggle SAML provider ${provider.display_name || provider.id}`}
                  onCheckedChange={handleToggleEnabled}
                />
                <span className="text-xs text-muted-foreground">{enabled ? "Enabled" : "Disabled"}</span>
              </div>
            </div>
            <SheetDescription className="text-xs">
              Configure identity provider settings, certificate validity, and attribute transformations.
            </SheetDescription>
          </SheetHeader>

          <form onSubmit={handleSave} className="flex-1 overflow-y-auto space-y-4 pt-4 pr-1 text-xs">
            {/* General Connection Settings */}
            <div className="space-y-3 p-3 rounded-lg border bg-card/60">
              <div className="font-semibold text-foreground">Connection Settings</div>

              <div className="space-y-1">
                <Label htmlFor="saml-edit-display-name" className="text-[11px]">
                  Provider Display Name
                </Label>
                <Input
                  id="saml-edit-display-name"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  className="h-7 text-xs"
                />
              </div>

              <div className="space-y-1">
                <Label htmlFor="saml-edit-sso-url" className="text-[11px]">
                  IdP SSO URL
                </Label>
                <Input
                  id="saml-edit-sso-url"
                  value={ssoUrl}
                  onChange={(e) => setSsoUrl(e.target.value)}
                  className="h-7 text-xs font-mono"
                />
              </div>

              <div className="space-y-1">
                <Label htmlFor="saml-edit-entity-id" className="text-[11px]">
                  IdP Entity ID
                </Label>
                <Input
                  id="saml-edit-entity-id"
                  value={provider.idp_entity_id}
                  readOnly
                  className="h-7 text-xs font-mono bg-muted/40"
                />
              </div>
            </div>

            {/* Certificate Management (P6-S7-T2) */}
            <div className="space-y-2.5 p-3 rounded-lg border bg-card/60">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-foreground flex items-center gap-1.5">
                  <ShieldCheck className="w-3.5 h-3.5 text-primary" />
                  IdP Signing Certificates
                </span>
                {certs.length > 0 && (
                  <span className="text-[10px] text-muted-foreground">{certs.length} installed</span>
                )}
              </div>

              {certs.length === 0 ? (
                <div className="p-2.5 rounded border border-dashed text-muted-foreground text-[11px] text-center">
                  Embedded certificate registered. Expiry:{" "}
                  {provider.idp_cert_expires_at
                    ? new Date(provider.idp_cert_expires_at).toLocaleDateString()
                    : "Not tracked"}
                </div>
              ) : (
                <div className="space-y-2">
                  {certs.map((c, i) => (
                    <div key={i} className="p-2 rounded border bg-background text-[11px] space-y-1 font-mono">
                      <div className="flex items-center justify-between">
                        <span className="font-semibold text-foreground truncate max-w-[200px]">
                          {c.subject || "IdP Certificate"}
                        </span>
                        {c.expired ? (
                          <Badge variant="destructive" className="text-[9px] gap-1 px-1 py-0">
                            <XCircle className="w-2.5 h-2.5" /> Expired
                          </Badge>
                        ) : c.expiring_soon ? (
                          <Badge
                            variant="outline"
                            className="text-[9px] text-amber-500 border-amber-500/30 gap-1 px-1 py-0"
                          >
                            <AlertTriangle className="w-2.5 h-2.5" /> Expiring in {c.days_until_expiry}d
                          </Badge>
                        ) : (
                          <Badge variant="default" className="bg-emerald-500 text-[9px] gap-1 px-1 py-0">
                            <CheckCircle2 className="w-2.5 h-2.5" /> Valid ({c.days_until_expiry}d left)
                          </Badge>
                        )}
                      </div>
                      <div className="text-muted-foreground text-[10px] flex items-center gap-1">
                        <Calendar className="w-3 h-3" />
                        Valid until: {new Date(c.not_after).toLocaleDateString()}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Attribute Mapping (P6-S7-T3) */}
            <div className="space-y-3 p-3 rounded-lg border bg-card/60">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-foreground">SAML Attribute Mapping</span>
                <span className="text-[10px] text-muted-foreground">IdP Claim ➔ Ego Trait</span>
              </div>

              <div className="space-y-1.5">
                {Object.entries(attributeMapping).map(([k, v]) => (
                  <div key={k} className="flex items-center gap-2 text-xs">
                    <span className="font-mono bg-muted px-2 py-1 rounded text-[11px] truncate w-1/2">{k}</span>
                    <span className="text-muted-foreground">➔</span>
                    <span className="font-mono bg-primary/10 text-primary px-2 py-1 rounded text-[11px] truncate w-1/2">
                      {v}
                    </span>
                    <Can perform="enterprise:write">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        aria-label={`Remove attribute mapping for ${k}`}
                        className="h-6 w-6 text-muted-foreground hover:text-destructive shrink-0"
                        onClick={() => removeMapping(k)}
                      >
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    </Can>
                  </div>
                ))}
              </div>

              {/* Add custom mapping row */}
              <Can perform="enterprise:write">
                <div className="flex items-center gap-2 pt-1">
                  <Input
                    aria-label="IdP Claim name"
                    placeholder="IdP Claim (e.g. mail)"
                    value={newKey}
                    onChange={(e) => setNewKey(e.target.value)}
                    className="h-7 text-xs font-mono"
                  />
                  <span className="text-muted-foreground">➔</span>
                  <Input
                    aria-label="Target Ego trait"
                    placeholder="Trait (e.g. email)"
                    value={newValue}
                    onChange={(e) => setNewValue(e.target.value)}
                    className="h-7 text-xs font-mono"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    aria-label="Add attribute mapping"
                    onClick={addMapping}
                    className="h-7 w-7 shrink-0"
                  >
                    <Plus className="w-3.5 h-3.5" />
                  </Button>
                </div>
              </Can>
            </div>

            <div className="flex items-center justify-between pt-4 border-t">
              <Can perform="enterprise:write">
                <Button
                  type="button"
                  variant="destructive"
                  size="sm"
                  onClick={() => setConfirmDeleteOpen(true)}
                  disabled={isDeleting}
                  className="gap-1.5 text-xs h-8"
                >
                  {isDeleting ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Trash2 className="w-3.5 h-3.5" />}
                  Delete Provider
                </Button>
              </Can>

              <div className="flex items-center gap-2 ml-auto">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => onOpenChange(false)}
                  className="h-8 text-xs"
                >
                  Cancel
                </Button>
                <Can perform="enterprise:write">
                  <Button type="submit" size="sm" disabled={isSaving} className="gap-1.5 text-xs h-8">
                    {isSaving ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Save className="w-3.5 h-3.5" />}
                    Save Changes
                  </Button>
                </Can>
              </div>
            </div>
          </form>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={confirmDeleteOpen}
        onOpenChange={setConfirmDeleteOpen}
        title="Delete SAML Provider"
        description={
          <>
            Permanently remove SAML identity provider <strong className="font-mono">{provider.id}</strong>? Federated
            sign-ins using this IdP will immediately fail.
          </>
        }
        confirmLabel="Delete Provider"
        variant="destructive"
        isLoading={isDeleting}
        onConfirm={handleDelete}
      />
    </>
  );
}
