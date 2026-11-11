import { Route, Routes } from "react-router-dom";
import BuildPage from "./pages/BuildPage";
import HomePage from "./pages/HomePage";

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<HomePage />} />
      <Route path="/build" element={<BuildPage />} />
      <Route path="*" element={<HomePage />} />
    </Routes>
  );
}
