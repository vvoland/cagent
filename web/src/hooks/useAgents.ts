import { useState, useEffect, useCallback, useMemo } from "react";

export interface Agent {
  name: string;
  description: string;
}

interface UseAgentsReturn {
  agents: Agent[];
  selectedAgent: string | null;
  isLoading: boolean;
  error: string | null;
  setSelectedAgent: (agent: string | null) => void;
  refreshAgents: () => Promise<void>;
}

export const useAgents = (): UseAgentsReturn => {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgent, setSelectedAgentState] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Memoize the fetch function to prevent recreation on every render
  const fetchAgents = useCallback(async (): Promise<void> => {
    try {
      setIsLoading(true);
      setError(null);
      
      const response = await fetch("/api/agents");
      
      if (!response.ok) {
        throw new Error(`Failed to fetch agents: ${response.statusText}`);
      }
      
      const data = await response.json() as Agent[];
      
      if (!Array.isArray(data)) {
        throw new Error("Invalid response format: expected array of agents");
      }
      
      setAgents(data);
      
      // Only set selected agent if we don't have one and there are agents available
      if (data.length > 0 && !selectedAgent) {
        setSelectedAgentState(data[0]?.name || null);
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "Failed to fetch agents";
      console.error("Failed to fetch agents:", error);
      setError(errorMessage);
      setAgents([]);
    } finally {
      setIsLoading(false);
    }
  }, [selectedAgent]); // Include selectedAgent to ensure we don't override user selection

  // Memoize the setSelectedAgent function
  const setSelectedAgent = useCallback((agent: string | null) => {
    setSelectedAgentState(agent);
    setError(null); // Clear any existing errors when changing agents
  }, []);

  // Memoize the refresh function
  const refreshAgents = useCallback((): Promise<void> => {
    return fetchAgents();
  }, [fetchAgents]);

  // Fetch agents on mount
  useEffect(() => {
    fetchAgents();
  }, [fetchAgents]);

  // Memoize the available agent names for quick lookups
  const agentNames = useMemo(() => agents.map(agent => agent.name), [agents]);

  // Validate selected agent still exists in the list
  useEffect(() => {
    if (selectedAgent && agentNames.length > 0 && !agentNames.includes(selectedAgent)) {
      // Selected agent no longer exists, select the first available one
      setSelectedAgentState(agentNames[0] || null);
    }
  }, [selectedAgent, agentNames]);

  // Memoize the return object to prevent unnecessary re-renders
  const returnValue = useMemo((): UseAgentsReturn => ({
    agents,
    selectedAgent,
    isLoading,
    error,
    setSelectedAgent,
    refreshAgents,
  }), [
    agents,
    selectedAgent,
    isLoading,
    error,
    setSelectedAgent,
    refreshAgents,
  ]);

  return returnValue;
};