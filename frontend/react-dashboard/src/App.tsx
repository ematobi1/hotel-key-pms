import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Login from './pages/Login';
import FrontDesk from './pages/FrontDesk';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = sessionStorage.getItem('access_token');
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/front-desk"
            element={
              <ProtectedRoute>
                <FrontDesk />
              </ProtectedRoute>
            }
          />
          <Route path="/" element={<Navigate to="/front-desk" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
