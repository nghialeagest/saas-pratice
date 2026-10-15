import React from "react";
import { Badge } from "@/components/ui/badge";

interface StatusBadgeProps {
  status: string;
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status }) => {
  const s = status.toUpperCase();

  switch (s) {
    case "PUBLISHED":
    case "ACTIVE":
    case "DONE":
      return <Badge variant="success">{s}</Badge>;
    case "DRAFT":
    case "PENDING":
    case "IN_PROGRESS":
      return <Badge variant="warning">{s}</Badge>;
    case "ARCHIVED":
    case "SUSPENDED":
    case "CANCELLED":
      return <Badge variant="destructive">{s}</Badge>;
    default:
      return <Badge variant="outline">{s}</Badge>;
  }
};
