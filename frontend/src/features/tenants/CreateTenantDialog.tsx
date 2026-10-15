import React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createTenantApi } from "@/api/tenants";
import { useTenant } from "@/hooks/use-tenant";
import { getLocalizedErrorMessage } from "@/lib/error-map";
import { useToast } from "@/components/ui/toast";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const tenantSchema = z.object({
  name: z.string().min(1, "Tên tổ chức/khách thuê không được để trống"),
  slug: z
    .string()
    .min(2, "Mã định danh (slug) ít nhất 2 ký tự")
    .regex(/^[a-z0-9-]+$/, "Slug chỉ bao gồm chữ thường không dấu, số và dấu gạch ngang (-)"),
});

type TenantFormData = z.infer<typeof tenantSchema>;

interface CreateTenantDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export const CreateTenantDialog: React.FC<CreateTenantDialogProps> = ({
  open,
  onOpenChange,
}) => {
  const queryClient = useQueryClient();
  const { refreshTenants } = useTenant();
  const { toast } = useToast();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<TenantFormData>({
    resolver: zodResolver(tenantSchema),
    defaultValues: {
      name: "",
      slug: "",
    },
  });

  const mutation = useMutation({
    mutationFn: (data: TenantFormData) =>
      createTenantApi({
        name: data.name,
        slug: data.slug,
      }),
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: ["tenants"] });
      await refreshTenants();
      toast({
        title: "Tạo khách thuê thành công",
        description: "Khách thuê mới đã được khởi tạo trong hệ thống.",
      });
      reset();
      onOpenChange(false);
    },
    onError: (err) => {
      toast({
        title: "Tạo khách thuê thất bại",
        description: getLocalizedErrorMessage(err),
        variant: "destructive",
      });
    },
  });

  const onSubmit = async (data: TenantFormData) => {
    await mutation.mutateAsync(data);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle>Tạo khách thuê (Tenant) mới</DialogTitle>
          <DialogDescription>
            Khởi tạo tổ chức mới trên phạm vi Host scope của hệ thống.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 py-2">
          <div className="space-y-2">
            <Label htmlFor="name">Tên tổ chức / Khách thuê *</Label>
            <Input
              id="name"
              placeholder="VD: Tập đoàn ABC"
              disabled={isSubmitting}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="slug">Slug định danh (Duy nhất) *</Label>
            <Input
              id="slug"
              placeholder="vd: tap-doan-abc"
              disabled={isSubmitting}
              {...register("slug")}
            />
            {errors.slug && (
              <p className="text-xs text-destructive">{errors.slug.message}</p>
            )}
          </div>

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Huỷ
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Đang tạo..." : "Tạo khách thuê"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
