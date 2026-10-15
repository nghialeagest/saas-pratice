import React, { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useNavigate, Navigate } from "react-router-dom";
import { useAuth } from "@/hooks/use-auth";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { useToast } from "@/components/ui/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Layers, AlertCircle, Lock } from "lucide-react";

const loginSchema = z.object({
  username_or_email: z.string().min(1, "Vui lòng nhập tên đăng nhập hoặc email"),
  password: z.string().min(1, "Vui lòng nhập mật khẩu"),
  tenant_id: z.string().optional(),
});

type LoginFormData = z.infer<typeof loginSchema>;

export const LoginPage: React.FC = () => {
  const { login, isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const { toast } = useToast();
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username_or_email: "",
      password: "",
      tenant_id: "",
    },
  });

  if (isAuthenticated) {
    return <Navigate to="/notes" replace />;
  }

  const onSubmit = async (data: LoginFormData) => {
    try {
      setErrorMessage(null);
      await login({
        username_or_email: data.username_or_email,
        password: data.password,
        tenant_id: data.tenant_id ? data.tenant_id.trim() : undefined,
      });
      toast({
        title: "Đăng nhập thành công",
        description: "Chào mừng bạn quay trở lại hệ thống.",
        variant: "default",
      });
      navigate("/notes");
    } catch (err: unknown) {
      const message = getLocalizedErrorMessage(err);
      setErrorMessage(message);
      toast({
        title: "Đăng nhập thất bại",
        description: message,
        variant: "destructive",
      });
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <Card className="w-full max-w-md shadow-lg border-muted">
        <CardHeader className="space-y-2 text-center">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow">
            <Layers className="h-6 w-6" />
          </div>
          <CardTitle className="text-2xl font-bold tracking-tight">
            Đăng nhập SaaS Base
          </CardTitle>
          <CardDescription className="text-sm text-muted-foreground">
            Hệ thống quản lý dịch vụ đa người thuê (Multi-Tenant)
          </CardDescription>
        </CardHeader>

        <form onSubmit={handleSubmit(onSubmit)}>
          <CardContent className="space-y-4">
            {errorMessage && (
              <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
                <AlertCircle className="h-4 w-4 shrink-0" />
                <span>{errorMessage}</span>
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="username_or_email">Tài khoản hoặc Email</Label>
              <Input
                id="username_or_email"
                placeholder="admin@example.com hoặc admin"
                disabled={isSubmitting}
                {...register("username_or_email")}
              />
              {errors.username_or_email && (
                <p className="text-xs text-destructive">
                  {errors.username_or_email.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Mật khẩu</Label>
              <Input
                id="password"
                type="password"
                placeholder="••••••••"
                disabled={isSubmitting}
                {...register("password")}
              />
              {errors.password && (
                <p className="text-xs text-destructive">
                  {errors.password.message}
                </p>
              )}
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="tenant_id">Mã khách thuê (Tenant ID - Tuỳ chọn)</Label>
                <span className="text-[11px] text-muted-foreground">Để trống = Host scope</span>
              </div>
              <Input
                id="tenant_id"
                placeholder="VD: tenant-alpha"
                disabled={isSubmitting}
                {...register("tenant_id")}
              />
              {errors.tenant_id && (
                <p className="text-xs text-destructive">
                  {errors.tenant_id.message}
                </p>
              )}
            </div>
            <div className="rounded-lg border border-primary/20 bg-primary/5 p-3 text-xs space-y-1.5">
              <div className="font-semibold text-primary flex items-center justify-between">
                <span>Tài khoản mẫu mặc định (Seed):</span>
                <button
                  type="button"
                  onClick={() => {
                    setValue("username_or_email", "admin");
                    setValue("password", "password123");
                    setValue("tenant_id", "tenant-default");
                  }}
                  className="text-[11px] underline text-primary hover:text-primary/80 font-medium cursor-pointer"
                >
                  Điền nhanh
                </button>
              </div>
              <div className="text-muted-foreground space-y-0.5">
                <div>• Tài khoản: <span className="font-mono text-foreground font-semibold">admin</span> hoặc <span className="font-mono text-foreground">admin@example.com</span></div>
                <div>• Mật khẩu: <span className="font-mono text-foreground font-semibold">password123</span></div>
                <div>• Tenant ID: <span className="font-mono text-foreground">tenant-default</span> (hoặc để trống)</div>
              </div>
            </div>
          </CardContent>

          <CardFooter className="pt-2">
            <Button
              type="submit"
              className="w-full gap-2"
              disabled={isSubmitting}
            >
              <Lock className="h-4 w-4" />
              {isSubmitting ? "Đang xác thực..." : "Đăng nhập"}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
};
