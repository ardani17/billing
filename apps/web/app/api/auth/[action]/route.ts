import { NextRequest, NextResponse } from "next/server";

const billingApiBaseUrl = process.env.BILLING_API_URL || "http://localhost:3001";

type RouteContext = {
  params: Promise<{ action: string }>;
};

function cookieMaxAge(expiresIn?: number) {
  if (!expiresIn || expiresIn < 1) return 60 * 60 * 24;
  return expiresIn;
}

async function authProxy(request: NextRequest, context: RouteContext) {
  const { action } = await context.params;
  const targetPath = `/api/v1/auth/${action}`;
  const method = request.method;
  const hasBody = !["GET", "HEAD"].includes(method);

  try {
    const response = await fetch(`${billingApiBaseUrl}${targetPath}`, {
      method,
      headers: {
        "Content-Type": "application/json",
        ...(request.cookies.get("ispboss_access_token")?.value
          ? { Authorization: `Bearer ${request.cookies.get("ispboss_access_token")?.value}` }
          : {}),
      },
      body: hasBody ? await request.text() : undefined,
      cache: "no-store",
    });

    const contentType = response.headers.get("content-type") || "";
    const body = contentType.includes("application/json") ? await response.json() : await response.text();
    const nextResponse = NextResponse.json(body, { status: response.status });

    if (response.ok && typeof body === "object" && body?.success && body?.data) {
      const accessToken = body.data.access_token;
      const refreshToken = body.data.refresh_token;
      const expiresIn = cookieMaxAge(body.data.expires_in);

      if (accessToken) {
        nextResponse.cookies.set("ispboss_access_token", accessToken, {
          httpOnly: true,
          sameSite: "lax",
          maxAge: expiresIn,
          path: "/",
        });
      }
      if (refreshToken) {
        nextResponse.cookies.set("ispboss_refresh_token", refreshToken, {
          httpOnly: true,
          sameSite: "lax",
          maxAge: 60 * 60 * 24 * 30,
          path: "/",
        });
      }
    }

    if (action === "logout") {
      nextResponse.cookies.delete("ispboss_access_token");
      nextResponse.cookies.delete("ispboss_refresh_token");
    }

    return nextResponse;
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "AUTH_API_ERROR",
          message: error instanceof Error ? error.message : "Auth API request failed",
        },
      },
      { status: 502 },
    );
  }
}

export async function POST(request: NextRequest, context: RouteContext) {
  return authProxy(request, context);
}
