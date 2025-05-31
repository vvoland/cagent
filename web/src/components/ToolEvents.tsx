import { useState } from "react";
import { Modal } from "./Modal";

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
      <div
        className="tool-pill tool-call-pill"
        onClick={() => setIsModalOpen(true)}
      >
        <span className="tool-icon">🛠️</span>
        <span className="tool-name">{name}</span>
      </div>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={`Tool Call: ${name}`}
      >
        <h4>Parameters:</h4>
        <pre className="tool-args">
          <code>{args}</code>
        </pre>
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
      <div
        className="tool-pill tool-result-pill"
        onClick={() => setIsModalOpen(true)}
      >
        <span className="tool-icon">✅</span>
        <span className="tool-name">{id}</span>
      </div>

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={`Tool Result: ${id}`}
      >
        <h4>Response:</h4>
        <div
          className="tool-content"
          style={{ overflowY: "auto", maxHeight: "500px" }}
        >
          <pre>{content}</pre>
        </div>
      </Modal>
    </>
  );
};
