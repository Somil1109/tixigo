import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type PropsWithChildren } from "react";
import { api, ApiError } from "../../lib/api";

export type User = { id: string; email: string; fullName: string; role: "customer" | "organiser" | "admin"; emailVerifiedAt?: string | null };
type Session = { data: { accessToken: string; user: User } };
type Value = { user: User | null; accessToken: string | null; loading: boolean; login(email: string, password: string): Promise<void>; register(fullName: string, email: string, password: string): Promise<void>; logout(): Promise<void>; request<T>(path: string, init?: RequestInit): Promise<T> };

const Context = createContext<Value | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const tokenRef = useRef<string | null>(null);
  const refreshRef = useRef<Promise<Session["data"]> | null>(null);

  const saveSession = useCallback((session: Session["data"] | null) => {
    tokenRef.current = session?.accessToken ?? null;
    setAccessToken(session?.accessToken ?? null);
    setUser(session?.user ?? null);
  }, []);

  const refreshSession = useCallback(async () => {
    if (!refreshRef.current) {
      refreshRef.current = api<Session>("/auth/refresh", { method: "POST" })
        .then(result => { saveSession(result.data); return result.data; })
        .finally(() => { refreshRef.current = null; });
    }
    return refreshRef.current;
  }, [saveSession]);

  useEffect(() => {
    refreshSession().catch(() => saveSession(null)).finally(() => setLoading(false));
  }, [refreshSession, saveSession]);

  const request = useCallback(async <T,>(path: string, init?: RequestInit): Promise<T> => {
    const send = (token: string | null) => api<T>(path, { ...init, headers: { ...init?.headers, ...(token ? { Authorization: `Bearer ${token}` } : {}) } });
    try {
      return await send(tokenRef.current);
    } catch (reason) {
      if (!(reason instanceof ApiError) || reason.status !== 401) throw reason;
      try {
        const session = await refreshSession();
        return await send(session.accessToken);
      } catch (refreshError) {
        saveSession(null);
        throw refreshError;
      }
    }
  }, [refreshSession, saveSession]);

  const value = useMemo<Value>(() => ({
    user,
    accessToken,
    loading,
    async login(email, password) {
      const result = await api<Session>("/auth/login", { method: "POST", body: JSON.stringify({ email, password }) });
      saveSession(result.data);
    },
    async register(fullName, email, password) {
      await api("/auth/register", { method: "POST", body: JSON.stringify({ fullName, email, password }) });
    },
    async logout() {
      try { await api("/auth/logout", { method: "POST" }); } finally { saveSession(null); }
    },
    request,
  }), [user, accessToken, loading, request, saveSession]);

  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useAuth() {
  const value = useContext(Context);
  if (!value) throw new Error("useAuth requires AuthProvider");
  return value;
}
