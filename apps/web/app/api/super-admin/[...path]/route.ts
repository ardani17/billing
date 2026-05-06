import { NextRequest, NextResponse } from "next/server";
import { createOptionalDevJwt, isProductionRuntime } from "../../../lib/dev-jwt";

const billingApiBaseUrl = process.env.BILLING_API_URL || "http://localhost:3001";

function cookieMaxAge(expiresIn?: number) {
  if (!expiresIn || expiresIn < 1) return 60 * 60 * 24;
  return expiresIn;
}

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

async function proxy(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  const query = request.nextUrl.search || "";
  const targetPath = path.join("/");
  const method = request.method;
  const hasBody = !["GET", "HEAD"].includes(method);
  const sessionToken = request.cookies.get("ispboss_access_token")?.value;
  const devToken = sessionToken ? null : createOptionalDevJwt({ role: "super_admin" });
  const authorizationToken = sessionToken || devToken;

  if (!authorizationToken) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "AUTH_REQUIRED",
          message: "Sesi super admin diperlukan",
        },
      },
      { status: 401 },
    );
  }

  try {
    const response = await fetch(`${billingApiBaseUrl}/api/v1/admin/${targetPath}${query}`, {
      method,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authorizationToken}`,
      },
      body: hasBody ? await request.text() : undefined,
      cache: "no-store",
    });

    const contentType = response.headers.get("content-type") || "";
    const body = contentType.includes("application/json") ? await response.json() : await response.text();

    if (contentType.includes("application/json")) {
      const nextResponse = NextResponse.json(body, { status: response.status });
      if (
        response.ok &&
        typeof body === "object" &&
        body?.success &&
        body?.data &&
        (targetPath === "impersonate" || targetPath === "stop-impersonate")
      ) {
        const accessToken = body.data.access_token;
        const refreshToken = body.data.refresh_token;
        if (accessToken) {
          nextResponse.cookies.set("ispboss_access_token", accessToken, {
            httpOnly: true,
            secure: isProductionRuntime(),
            sameSite: "lax",
            maxAge: cookieMaxAge(body.data.expires_in),
            path: "/",
          });
        }
        if (refreshToken) {
          nextResponse.cookies.set("ispboss_refresh_token", refreshToken, {
            httpOnly: true,
            secure: isProductionRuntime(),
            sameSite: "lax",
            maxAge: 60 * 60 * 24 * 30,
            path: "/",
          });
        }
      }
      return nextResponse;
    }

    return new NextResponse(body, {
      status: response.status,
      headers: { "Content-Type": contentType || "text/plain; charset=utf-8" },
    });
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "SUPER_ADMIN_API_ERROR",
          message: error instanceof Error ? error.message : "Super admin API request failed",
        },
      },
      { status: 502 },
    );
  }
}

export async function GET(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

export async function POST(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

export async function PUT(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}
