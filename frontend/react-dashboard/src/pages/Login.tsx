import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router';
import { setAuthTokens } from '../api/client';

export default function Login() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const keycloakUrl = import.meta.env.VITE_KEYCLOAK_URL;
  const realm = import.meta.env.VITE_KEYCLOAK_REALM ?? 'hotel-key';
  const clientId = import.meta.env.VITE_KEYCLOAK_CLIENT_ID ?? 'pms-dashboard';

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const params = new URLSearchParams({
        grant_type: 'password',
        client_id: clientId,
        username,
        password,
      });

      const res = await fetch(
        `${keycloakUrl}/realms/${realm}/protocol/openid-connect/token`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: params.toString(),
        }
      );

      if (!res.ok) {
        const data = await res.json();
        setError(data.error_description ?? 'Login failed');
        return;
      }

      const data = await res.json();
      const tenantId = parseTenantFromToken(data.access_token);
      setAuthTokens(data.access_token, tenantId);
      navigate('/front-desk');
    } catch {
      setError('Network error — please try again');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh', background: '#f5f5f5' }}>
      <form
        onSubmit={handleSubmit}
        style={{ background: '#fff', padding: '2rem', borderRadius: '8px', minWidth: '320px', boxShadow: '0 2px 8px rgba(0,0,0,0.1)' }}
        aria-label="Sign in to Hotel Key PMS"
      >
        <h1 style={{ marginBottom: '1.5rem', fontSize: '1.5rem' }}>Hotel Key PMS</h1>
        {error && (
          <p role="alert" style={{ color: '#d32f2f', marginBottom: '1rem' }}>{error}</p>
        )}
        <label htmlFor="username" style={{ display: 'block', marginBottom: '0.25rem' }}>Username</label>
        <input
          id="username"
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          required
          autoComplete="username"
          style={{ width: '100%', padding: '0.5rem', marginBottom: '1rem', boxSizing: 'border-box' }}
        />
        <label htmlFor="password" style={{ display: 'block', marginBottom: '0.25rem' }}>Password</label>
        <input
          id="password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          autoComplete="current-password"
          style={{ width: '100%', padding: '0.5rem', marginBottom: '1.5rem', boxSizing: 'border-box' }}
        />
        <button
          type="submit"
          disabled={loading}
          style={{ width: '100%', padding: '0.75rem', background: '#1976d2', color: '#fff', border: 'none', borderRadius: '4px', cursor: loading ? 'not-allowed' : 'pointer' }}
        >
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </main>
  );
}

function parseTenantFromToken(token: string): string {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.tenant_id ?? payload.sub ?? '';
  } catch {
    return '';
  }
}
