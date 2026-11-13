import { createContext, useContext, useState } from "react";
import { createAuthClient } from "../api/authApi";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [state, setState] = useState({ authenticated: false, user: null, profile: null });
  const [client] = useState(() => createAuthClient(setState));

  return (
    <AuthContext.Provider value={{ ...state, client }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
