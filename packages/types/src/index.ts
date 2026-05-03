// Tipe respons standar API untuk semua endpoint ISPBoss.
export interface ApiResponse<T = unknown> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
  };
}
