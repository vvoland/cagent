import { useState, useEffect } from "react";

export interface Agent {
  name: string;
  description: string;
}

export const useAgents = () => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchAgents = async () => {
      try {
        const response = await fetch("/api/agents");
        const data = await response.json();
        setAgents(data);
        if (data.length > 0) {
          setSelectedAgent(data[0].name);
        }
      } catch (error) {
        console.error("Failed to fetch agents:", error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchAgents();
  }, []);

  return {
    agents,
    selectedAgent,
    setSelectedAgent,
    isLoading,
  };
};
