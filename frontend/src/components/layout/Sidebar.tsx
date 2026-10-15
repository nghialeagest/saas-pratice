import React from "react";
import { NavLink } from "react-router-dom";
import { FileText, CheckSquare, Building2, Layers, Users, ShieldCheck } from "lucide-react";
import { cn } from "@/lib/utils";
import { Can } from "@/components/shared/Can";

interface NavItem {
  name: string;
  to: string;
  icon: React.ElementType;
  resource?: string;
  action?: string;
}

const navItems: NavItem[] = [
  { name: "Ghi chú (Notes)", to: "/notes", icon: FileText, resource: "note.note", action: "read" },
  { name: "Công việc (Tasks)", to: "/tasks", icon: CheckSquare, resource: "task.task", action: "read" },
  { name: "Người dùng (Users)", to: "/users", icon: Users, resource: "user.user", action: "read" },
  { name: "Vai trò & Quyền", to: "/roles", icon: ShieldCheck, resource: "user.role", action: "read" },
  { name: "Khách thuê (Tenants)", to: "/tenants", icon: Building2, resource: "tenant.tenant", action: "read" },
];

export const Sidebar: React.FC = () => {
  return (
    <aside className="w-64 border-r bg-card min-h-screen flex flex-col">
      <div className="h-16 flex items-center px-6 border-b gap-3">
        <div className="p-2 bg-primary text-primary-foreground rounded-lg shadow">
          <Layers className="h-5 w-5" />
        </div>
        <div>
          <h1 className="font-bold text-base tracking-tight leading-none">SaaS Base</h1>
          <p className="text-xs text-muted-foreground mt-1">Multi-tenant Cloud</p>
        </div>
      </div>

      <nav className="flex-1 px-4 py-6 space-y-1">
        <div className="text-xs font-semibold text-muted-foreground uppercase px-2 mb-2 tracking-wider">
          Menu chính
        </div>
        {navItems.map((item) => {
          const Icon = item.icon;
          const link = (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors",
                  isActive
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:bg-muted hover:text-foreground"
                )
              }
            >
              <Icon className="h-4 w-4 shrink-0" />
              <span>{item.name}</span>
            </NavLink>
          );

          if (item.resource && item.action) {
            return (
              <Can key={item.to} resource={item.resource} action={item.action}>
                {link}
              </Can>
            );
          }

          return link;
        })}
      </nav>

      <div className="p-4 border-t text-xs text-muted-foreground">
        <div>Platform v1.0.0</div>
        <div className="text-[10px] text-muted-foreground/80 mt-0.5">Isolated Multi-Tenancy</div>
      </div>
    </aside>
  );
};
