import React, { useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { login, clearError } from '../../store/slices/authSlice';
import { RootState } from '../../store';
import { Navigate, Link } from 'react-router-dom';

const Login: React.FC = () => {
  const dispatch = useDispatch();
  const { isAuthenticated, isLoading, error } = useSelector((state: RootState) => state.auth);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitted, setSubmitted] = useState(false);

  if (isAuthenticated) return <Navigate to="/" replace />;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    await dispatch(login({ email, password }));
  };

  return (
    <div className="auth-container" style={{ maxWidth: 400, margin: '80px auto', padding: 32, background: '#fff', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.07)' }}>
      <h2 style={{ marginBottom: 24 }}>Sign In</h2>
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: 16 }}>
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => { setEmail(e.target.value); if (error) dispatch(clearError()); }}
            required
            autoFocus
            style={{ width: '100%', padding: 8, marginTop: 4 }}
          />
        </div>
        <div style={{ marginBottom: 16 }}>
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => { setPassword(e.target.value); if (error) dispatch(clearError()); }}
            required
            style={{ width: '100%', padding: 8, marginTop: 4 }}
          />
        </div>
        {error && <div style={{ color: 'red', marginBottom: 12 }}>{error}</div>}
        <button type="submit" disabled={isLoading || !email || !password} style={{ width: '100%', padding: 12, background: 'var(--color-primary)', color: '#fff', border: 'none', borderRadius: 6, fontWeight: 600 }}>
          {isLoading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>
      <div style={{ marginTop: 16, textAlign: 'center' }}>
        <span>Don't have an account? </span>
        <Link to="/register">Sign Up</Link>
      </div>
    </div>
  );
};

export default Login;
