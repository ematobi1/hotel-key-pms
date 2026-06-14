import { useQuery } from '@tanstack/react-query';
import { reservationsApi, clearAuthTokens } from '../api/client';
import { useNavigate } from 'react-router-dom';
import { format } from 'date-fns';

interface Reservation {
  id: string;
  status: string;
  arrival_date: string;
  departure_date: string;
  num_adults: number;
  num_children: number;
  channel: string;
  currency: string;
  total_amount?: number;
}

export default function FrontDesk() {
  const navigate = useNavigate();
  const hotelId = sessionStorage.getItem('hotel_id') ?? '';

  const { data, isLoading, error } = useQuery({
    queryKey: ['reservations', hotelId],
    queryFn: () => reservationsApi.list({ hotel_id: hotelId, limit: 50 }),
    enabled: Boolean(hotelId),
    refetchInterval: 60_000,
  });

  function handleSignOut() {
    clearAuthTokens();
    navigate('/login');
  }

  const reservations: Reservation[] = data?.data?.data ?? [];

  return (
    <div style={{ fontFamily: 'system-ui, sans-serif', padding: '1.5rem' }}>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
        <h1 style={{ fontSize: '1.25rem', fontWeight: 600 }}>Front Desk</h1>
        <button
          type="button"
          onClick={handleSignOut}
          style={{ padding: '0.4rem 1rem', cursor: 'pointer' }}
        >
          Sign out
        </button>
      </header>

      {isLoading && <p aria-live="polite">Loading reservations…</p>}
      {error && <p role="alert" style={{ color: '#d32f2f' }}>Failed to load reservations</p>}

      {!isLoading && (
        <table style={{ width: '100%', borderCollapse: 'collapse' }} aria-label="Today's reservations">
          <thead>
            <tr style={{ background: '#f5f5f5' }}>
              <th style={th}>ID</th>
              <th style={th}>Arrival</th>
              <th style={th}>Departure</th>
              <th style={th}>Guests</th>
              <th style={th}>Status</th>
              <th style={th}>Channel</th>
              <th style={th}>Total</th>
            </tr>
          </thead>
          <tbody>
            {reservations.length === 0 && (
              <tr><td colSpan={7} style={{ textAlign: 'center', padding: '1rem' }}>No reservations found</td></tr>
            )}
            {reservations.map((rv) => (
              <tr key={rv.id} style={{ borderBottom: '1px solid #e0e0e0' }}>
                <td style={td}><code>{rv.id.slice(0, 8)}</code></td>
                <td style={td}>{format(new Date(rv.arrival_date), 'yyyy-MM-dd')}</td>
                <td style={td}>{format(new Date(rv.departure_date), 'yyyy-MM-dd')}</td>
                <td style={td}>{rv.num_adults}A / {rv.num_children}C</td>
                <td style={td}><StatusBadge status={rv.status} /></td>
                <td style={td}>{rv.channel}</td>
                <td style={td}>{rv.total_amount != null ? `${rv.currency} ${rv.total_amount.toLocaleString()}` : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    confirmed:   '#1976d2',
    checked_in:  '#388e3c',
    checked_out: '#757575',
    cancelled:   '#d32f2f',
    no_show:     '#f57c00',
    tentative:   '#7b1fa2',
  };
  return (
    <span style={{
      display: 'inline-block',
      padding: '0.2rem 0.5rem',
      borderRadius: '12px',
      background: colors[status] ?? '#9e9e9e',
      color: '#fff',
      fontSize: '0.75rem',
      fontWeight: 500,
    }}>
      {status.replace('_', ' ')}
    </span>
  );
}

const th: React.CSSProperties = { padding: '0.5rem 0.75rem', textAlign: 'left', fontWeight: 600 };
const td: React.CSSProperties = { padding: '0.5rem 0.75rem' };
