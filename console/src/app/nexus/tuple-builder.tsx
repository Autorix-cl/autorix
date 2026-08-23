import * as React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Loader2, Plus } from "lucide-react";

export interface TupleData {
  objectType: string;
  objectId: string;
  relation: string;
  subjectType: string;
  subjectId: string;
  subjectRelation?: string;
}

export interface TupleBuilderProps {
  onSubmit: (data: TupleData) => Promise<void>;
  onCancel: () => void;
}

export function TupleBuilder({ onSubmit, onCancel }: TupleBuilderProps) {
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  
  const [objectType, setObjectType] = React.useState("");
  const [objectId, setObjectId] = React.useState("");
  const [relation, setRelation] = React.useState("");
  
  const [subjectType, setSubjectType] = React.useState("");
  const [subjectId, setSubjectId] = React.useState("");
  const [subjectRelation, setSubjectRelation] = React.useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await onSubmit({
        objectType,
        objectId,
        relation,
        subjectType,
        subjectId,
        subjectRelation: subjectRelation || undefined,
      });
      // Reset form could go here, but usually parent closes it
    } catch (err) {
      console.error(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="grid grid-cols-2 gap-4">
        {/* Object Side */}
        <div className="space-y-4 p-4 border rounded-md bg-muted/10">
          <h4 className="text-sm font-semibold mb-2 text-foreground">Object</h4>
          <div className="space-y-1">
            <label htmlFor="objectType" className="text-xs font-medium text-muted-foreground">Object Type *</label>
            <Input id="objectType" value={objectType} onChange={(e) => setObjectType(e.target.value)} placeholder="e.g. document" required className="h-8 text-sm" />
          </div>
          <div className="space-y-1">
            <label htmlFor="objectId" className="text-xs font-medium text-muted-foreground">Object ID *</label>
            <Input id="objectId" value={objectId} onChange={(e) => setObjectId(e.target.value)} placeholder="e.g. doc_1" required className="h-8 text-sm" />
          </div>
        </div>

        {/* Relation */}
        <div className="col-span-2 sm:col-span-1 space-y-1">
          <label htmlFor="relation" className="text-xs font-medium text-muted-foreground">Relation *</label>
          <Input id="relation" value={relation} onChange={(e) => setRelation(e.target.value)} placeholder="e.g. viewer" required className="h-8 text-sm border-dashed" />
        </div>

        {/* Subject Side */}
        <div className="space-y-4 p-4 border rounded-md bg-muted/10 col-span-2">
          <h4 className="text-sm font-semibold mb-2 text-foreground">Subject (User or Set)</h4>
          <div className="grid grid-cols-3 gap-2">
            <div className="space-y-1">
              <label htmlFor="subjectType" className="text-xs font-medium text-muted-foreground">Subject Type *</label>
              <Input id="subjectType" value={subjectType} onChange={(e) => setSubjectType(e.target.value)} placeholder="e.g. user" required className="h-8 text-sm" />
            </div>
            <div className="space-y-1">
              <label htmlFor="subjectId" className="text-xs font-medium text-muted-foreground">Subject ID *</label>
              <Input id="subjectId" value={subjectId} onChange={(e) => setSubjectId(e.target.value)} placeholder="e.g. alice" required className="h-8 text-sm" />
            </div>
            <div className="space-y-1">
              <label htmlFor="subjectRelation" className="text-xs font-medium text-muted-foreground">Subject Relation</label>
              <Input id="subjectRelation" value={subjectRelation} onChange={(e) => setSubjectRelation(e.target.value)} placeholder="e.g. member (optional)" className="h-8 text-sm" />
            </div>
          </div>
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-4">
        <Button type="button" variant="ghost" onClick={onCancel} disabled={isSubmitting}>
          Cancel
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Plus className="h-4 w-4 mr-2" />}
          Create Tuple
        </Button>
      </div>
    </form>
  );
}
