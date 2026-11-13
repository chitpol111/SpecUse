import { createContext, useContext, useEffect, useRef, useState } from "react";
import { createAuthClient, readStoredSession } from "../api/authApi";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [initial] = useState(readStoredSession);
  const [state, setState] = useState({ authenticated: Boolean(initial.session), user: null, profile: null });
  const [storageWarning, setStorageWarning] = useState(initial.warning);
  const [restoreError, setRestoreError] = useState("");
  const restorePromise = useRef(null);
  const [client] = useState(() => createAuthClient((next) => {
    setState(next);
    if (!next.authenticated || next.profile) setRestoreError("");
  }, { initialSession: initial.session, onStorageStatus: setStorageWarning }));

  useEffect(() => {
    if (!initial.session) return undefined;
    let active = true;
    if (!restorePromise.current) restorePromise.current = client.restoreProfile();
    restorePromise.current.catch((error) => {
      if (active) setRestoreError(`กู้สถานะเข้าสู่ระบบไม่สำเร็จ: ${error.message}`);
    });
    return () => { active = false; };
  }, [client, initial.session]);

  return (
    <AuthContext.Provider value={{ ...state, client, sessionWarning: storageWarning || restoreError }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
