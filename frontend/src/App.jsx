import { Route, Routes } from "react-router-dom";
import { AuthProvider } from "./auth/AuthContext";
import AuthPage from "./pages/AuthPage";
import AdminProductsPage from "./pages/AdminProductsPage";
import BuildPage from "./pages/BuildPage";
import HomePage from "./pages/HomePage";
import ProfilePage from "./pages/ProfilePage";

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/build" element={<BuildPage />} />
        <Route path="/login" element={<AuthPage key="login" mode="login" />} />
        <Route path="/register" element={<AuthPage key="register" mode="register" />} />
        <Route path="/profile" element={<ProfilePage />} />
        <Route path="/admin/products" element={<AdminProductsPage />} />
        <Route path="*" element={<HomePage />} />
      </Routes>
    </AuthProvider>
  );
}
