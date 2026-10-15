import React from "react";
import { useAuth } from "@/hooks/use-auth";
import { useTenant } from "@/hooks/use-tenant";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { LogOut, Building, User as UserIcon } from "lucide-react";

export const Header: React.FC = () => {
  const { user, logout } = useAuth();
  const { tenantId, setTenantId, tenants, isLoadingTenants } = useTenant();
  console.log(user)

  return (
    <header className="h-16 border-b bg-card px-6 flex items-center justify-between shadow-sm">
      {/* Tenant Switcher */}
      <div className="flex items-center gap-3">
        <Building className="h-4 w-4 text-muted-foreground" />
        <span className="text-xs font-medium text-muted-foreground">Không gian làm việc:</span>
        <div className="w-56">
          <Select
            value={tenantId || "host"}
            onValueChange={(val) => {
              if (val === "host") {
                setTenantId("");
              } else {
                setTenantId(val);
              }
            }}
            disabled={isLoadingTenants}
          >
            <SelectTrigger className="h-8 text-xs">
              <SelectValue placeholder="Chọn Tenant..." />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="host">
                <span className="font-semibold text-primary">Toàn hệ thống (Host Scope)</span>
              </SelectItem>
              {tenants.map((t) => (
                <SelectItem key={t.id} value={t.id}>
                  {t.name} ({t.slug})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* User Info & Actions */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <div className="p-1.5 rounded-full bg-muted">
            <UserIcon className="h-3.5 w-3.5" />
          </div>
          <span className="font-medium text-foreground">{user?.username || "Người dùng"}</span>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={logout}
          className="h-8 text-xs gap-1.5 hover:bg-destructive/10 hover:text-destructive hover:border-destructive/30"
        >
          <LogOut className="h-3.5 w-3.5" />
          <span>Đăng xuất</span>
        </Button>
      </div>
    </header>
  );
};
