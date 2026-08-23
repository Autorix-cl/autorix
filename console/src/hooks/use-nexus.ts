import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as api from "@/lib/api/nexus";

export function useNexusSchema() {
  return useQuery({
    queryKey: ["nexus", "schema"],
    queryFn: api.fetchSchema,
  });
}

export function useSaveNexusSchema() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.updateSchema,
    onSuccess: (newSchema) => {
      queryClient.setQueryData(["nexus", "schema"], newSchema);
    },
  });
}

export function useNexusTuples() {
  return useQuery({
    queryKey: ["nexus", "tuples"],
    queryFn: api.fetchTuples,
  });
}

export function useAddNexusTuple() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.addTuple,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["nexus", "tuples"] });
    },
  });
}

export function useDeleteNexusTuples() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.deleteTuples,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["nexus", "tuples"] });
    },
  });
}

export function useNexusCheck() {
  return useMutation({
    mutationFn: api.checkAccess,
  });
}
