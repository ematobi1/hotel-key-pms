import { BrowserRouter, Routes, Route, Navigate } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient();

function HousekeepingBoard() {
  return (
    <main style={{ padding: '1rem', fontFamily: 'system-ui, sans-serif' }}>
      <h1 style={{ fontSize: '1.2rem', marginBottom: '1rem' }}>Housekeeping Tasks</h1>
      {/* TODO: list today's tasks from housekeeping service */}
      <p style={{ color: '#757575' }}>Loading tasks…</p>
    </main>
  );
}

function MaintenanceBoard() {
  return (
    <main style={{ padding: '1rem', fontFamily: 'system-ui, sans-serif' }}>
      <h1 style={{ fontSize: '1.2rem', marginBottom: '1rem' }}>Maintenance Work Orders</h1>
      {/* TODO: list open work orders from maintenance service */}
      <p style={{ color: '#757575' }}>Loading work orders…</p>
    </main>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/housekeeping" element={<HousekeepingBoard />} />
          <Route path="/maintenance" element={<MaintenanceBoard />} />
          <Route path="/" element={<Navigate to="/housekeeping" replace />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
