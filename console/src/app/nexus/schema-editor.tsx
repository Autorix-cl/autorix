import * as React from "react";
import { Button } from "@/components/ui/button";
import { CodeEditor } from "@/components/ui/code-editor";
import { Loader2, Edit2, Save, FileCode } from "lucide-react";

export interface SchemaEditorProps {
  schema: string;
  onSave: (newSchema: string) => Promise<void>;
}

export function SchemaEditor({ schema, onSave }: SchemaEditorProps) {
  const [isEditing, setIsEditing] = React.useState(false);
  const [isSaving, setIsSaving] = React.useState(false);
  const [currentSchema, setCurrentSchema] = React.useState(schema);

  // Sync prop changes when not editing
  React.useEffect(() => {
    if (!isEditing) {
      setCurrentSchema(schema);
    }
  }, [schema, isEditing]);

  const handleSave = async () => {
    setIsSaving(true);
    try {
      await onSave(currentSchema);
      setIsEditing(false);
    } catch (e) {
      // Allow parent to handle/toast the error, but we stay in edit mode
      console.error(e);
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancel = () => {
    setIsEditing(false);
    setCurrentSchema(schema); // Revert changes
  };

  return (
    <div className="flex flex-col h-full rounded-md border bg-card text-card-foreground shadow-sm">
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-2">
          <FileCode className="h-4 w-4 text-primary" />
          <h3 className="font-semibold text-sm">Authorization Schema</h3>
        </div>
        <div className="flex items-center gap-2">
          {!isEditing ? (
            <Button size="sm" variant="outline" onClick={() => setIsEditing(true)}>
              <Edit2 className="h-4 w-4 mr-2" />
              Edit Schema
            </Button>
          ) : (
            <>
              <Button size="sm" variant="ghost" onClick={handleCancel} disabled={isSaving}>
                Cancel
              </Button>
              <Button size="sm" onClick={handleSave} disabled={isSaving}>
                {isSaving ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Save className="h-4 w-4 mr-2" />}
                Save & Validate
              </Button>
            </>
          )}
        </div>
      </div>
      <div className="flex-1 relative min-h-[400px]">
        {/* We use java as a close approximation for Zanzibar DSL syntax highlighting in basic CodeMirror */}
        <CodeEditor
          value={currentSchema}
          onChange={setCurrentSchema}
          language="json"
          readOnly={!isEditing}
          height="100%"
        />
        {!isEditing && (
          <div className="absolute inset-0 bg-transparent" title="Read Only Mode. Click Edit to change." />
        )}
      </div>
    </div>
  );
}
