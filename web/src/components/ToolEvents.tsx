import { useState, useCallback, memo } from "react";
import { Modal } from "./Modal";
import { Button } from "./ui/button";
import { cn } from "../lib/utils";
import { Code, CheckCircle, Copy } from "lucide-react";
import { SimpleToolChip, SimpleToolChipGroup } from "./SimpleToolChip";

interface ToolCallEventProps {
  name: string;
  args: string;
}

interface ToolResultEventProps {
  id: string;
  content: string;
}

const formatJSON = (jsonString: string): string => {
  try {
    const parsed = JSON.parse(jsonString);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return jsonString;
  }
};

const CopyButton = memo<{ text: string; className?: string }>(({ text, className }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy text:', error);
    }
  }, [text]);

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={handleCopy}
      className={cn("h-6 w-6 p-0", className)}
      aria-label={copied ? "Copied!" : "Copy to clipboard"}
    >
      {copied ? (
        <CheckCircle className="h-3 w-3 text-green-500" />
      ) : (
        <Copy className="h-3 w-3" />
      )}
    </Button>
  );
});

CopyButton.displayName = 'CopyButton';

export const ToolCallEvent = memo<ToolCallEventProps>(({ name, args }) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  
  const closeModal = useCallback(() => setIsModalOpen(false), []);

  const formattedArgs = formatJSON(args);

  return (
    <SimpleToolChipGroup className="mx-2 lg:mx-3">
      <SimpleToolChip
        name={name}
        args={args}
        variant="call"
        className="transition-all hover:shadow-md hover:scale-105 active:scale-100"
      />

      {/* Fallback modal for detailed view - can be removed if chip expanded view is sufficient */}
      <Modal
        isOpen={isModalOpen}
        onClose={closeModal}
        title={`Tool Call: ${name}`}
        className="max-w-2xl"
      >
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h4 className="font-semibold text-sm flex items-center gap-2">
              <Code className="h-4 w-4" />
              Parameters
            </h4>
            <CopyButton text={formattedArgs} />
          </div>
          
          <div className={cn(
            "p-4 rounded-lg border",
            "bg-secondary/50 dark:bg-secondary/30",
            "border-border/50 dark:border-border/30",
            "overflow-x-auto"
          )}>
            <pre className="text-sm font-mono whitespace-pre-wrap break-words">
              <code className="text-foreground">{formattedArgs}</code>
            </pre>
          </div>
          
          <div className="text-xs text-muted-foreground bg-muted/30 p-2 rounded border border-muted">
            <strong>Tool:</strong> {name}
          </div>
        </div>
      </Modal>
    </SimpleToolChipGroup>
  );
});

ToolCallEvent.displayName = 'ToolCallEvent';

export const ToolResultEvent = memo<ToolResultEventProps>(({ id, content }) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  
  const closeModal = useCallback(() => setIsModalOpen(false), []);

  // Extract tool name from content or use a default
  const toolName = "result"; // You might want to pass this as a prop or extract it

  return (
    <SimpleToolChipGroup className="mx-2 lg:mx-3">
      <SimpleToolChip
        name={toolName}
        result={content}
        variant="result"
        className="transition-all hover:shadow-md hover:scale-105 active:scale-100"
      />

      {/* Fallback modal for detailed view - can be removed if chip expanded view is sufficient */}
      <Modal
        isOpen={isModalOpen}
        onClose={closeModal}
        title={`Tool Result: ${id}`}
        className="max-w-4xl"
      >
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h4 className="font-semibold text-sm flex items-center gap-2">
              <CheckCircle className="h-4 w-4" />
              Response
            </h4>
            <CopyButton text={content} />
          </div>
          
          <div className="max-h-[60vh] overflow-y-auto">
            <div className={cn(
              "p-4 rounded-lg border",
              "bg-secondary/50 dark:bg-secondary/30",
              "border-border/50 dark:border-border/30",
              "overflow-x-auto"
            )}>
              <pre className="text-sm font-mono whitespace-pre-wrap break-words">
                <code className="text-foreground">{content}</code>
              </pre>
            </div>
          </div>
          
          <div className="text-xs text-muted-foreground bg-muted/30 p-2 rounded border border-muted">
            <strong>Tool ID:</strong> {id}
          </div>
        </div>
      </Modal>
    </SimpleToolChipGroup>
  );
});

ToolResultEvent.displayName = 'ToolResultEvent';