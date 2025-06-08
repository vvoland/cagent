import { useState } from "react";
import { Modal } from "./Modal";
import { Button } from "./ui/button";
import { cn } from "../lib/utils";

export const ToolCallEvent = ({
  name,
  args,
}: {
  name: string;
  args: string;
}) => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        className="inline-flex items-center gap-2 m-3"
        onClick={() => setIsModalOpen(true)}
      >
        <span className="text-lg">🛠️</span>
        <span className="font-medium">tool: {name}</span>
      </Button>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={`Tool Call: ${name}`}
      >
        <div className="space-y-4">
          <h4 className="font-semibold text-sm">Parameters:</h4>
          <div className={cn("p-4 rounded-lg", "text-sm font-mono")}>
            <code>{args}</code>
          </div>
        </div>
      </Modal>
    </>
  );
};

export const ToolResultEvent = ({
  id,
  content,
}: {
  id: string;
  content: string;
}) => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        className="inline-flex items-center gap-2"
        onClick={() => setIsModalOpen(true)}
      >
        <span className="text-lg">✅</span>
        <span className="font-medium">result: {id}</span>
      </Button>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={`Tool Result: ${id}`}
      >
        <div className="space-y-4">
          <h4 className="font-semibold text-sm">Response:</h4>
          <div className="max-h-[500px] overflow-y-auto">
            <div
              className={cn(
                "p-4 rounded-lg overflow-x-auto",
                "text-sm font-mono "
              )}
            >
              {content}
            </div>
          </div>
        </div>
      </Modal>
    </>
  );
};
