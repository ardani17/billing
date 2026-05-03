// Utility untuk menggabungkan class names.
// Akan digunakan oleh komponen shadcn/ui setelah inisialisasi.
export function cn(...classes: (string | undefined | null | false)[]) {
  return classes.filter(Boolean).join(" ");
}
