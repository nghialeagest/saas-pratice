import React from "react";
import { useAuth } from "@/hooks/use-auth";

interface CanProps {
  resource: string;
  action: string;
  fallback?: React.ReactNode;
  children: React.ReactNode;
}

export const Can: React.FC<CanProps> = ({
  resource,
  action,
  fallback = null,
  children,
}) => {
  const { can } = useAuth();

  if (!can(resource, action)) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};
