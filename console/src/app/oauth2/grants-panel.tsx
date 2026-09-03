"use client";

import * as React from "react";
import { Users, RefreshCw } from "lucide-react";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
import { useApiQuery } from "@/lib/query/use-api-query";
import { fetchAndParse } from "@/lib/api/schema";
import { oauth2GrantListSchema, type OAuth2Grant } from "@/lib/api/schemas/oauth2";
import { LoadingState } from "@/components/state/loading-state";
import { EmptyState } from "@/components/state/empty-state";

export function GrantsPanel() {
  const {
    data: grantsRaw,
    isLoading,
    isError,
    refetch,
  } = useApiQuery(["oauth2-grants"], () =>
    fetchAndParse<OAuth2Grant[]>("/api/oauth2/grants", oauth2GrantListSchema)
  );

  const grants = grantsRaw ?? [];

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle className="text-base font-semibold flex items-center gap-2">
            <Users className="w-4 h-4 text-primary" />
            Consents & Authorization Grants
          </CardTitle>
          <CardDescription>
            Active user consents granted to OAuth2 clients and authorized scope delegations.
          </CardDescription>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          <RefreshCw className="w-4 h-4" />
        </Button>
      </CardHeader>
      <CardContent>
        {isLoading && <LoadingState label="Loading active grants..." />}

        {isError && (
          <div className="p-4 rounded border border-destructive/20 bg-destructive/10 text-destructive text-sm">
            Failed to load grants from Janus.
          </div>
        )}

        {!isLoading && !isError && grants.length === 0 && (
          <EmptyState
            title="No active grants"
            description="User authorization code consents and refresh grants will appear here as users authorize clients."
          />
        )}

        {!isLoading && grants.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Grant ID</TableHead>
                <TableHead>Subject</TableHead>
                <TableHead>Client ID</TableHead>
                <TableHead>Granted Scopes</TableHead>
                <TableHead>Granted At</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {grants.map((g) => (
                <TableRow key={g.grant_id}>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {g.grant_id}
                  </TableCell>
                  <TableCell className="font-medium text-sm">
                    {g.subject}
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {g.client_id}
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {g.scopes?.map((s) => (
                        <Badge key={s} variant="secondary" className="font-mono text-xs">
                          {s}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {new Date(g.created_at).toLocaleString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
