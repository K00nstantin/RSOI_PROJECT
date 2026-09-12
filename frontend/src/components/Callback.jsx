import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

function Callback() {
    const navigate = useNavigate();

    useEffect(() => {
        const hash = window.location.hash;
        if (!hash) {
            navigate('/login');
            return;
        }

        const params = new URLSearchParams(hash.slice(1));
        const token = params.get('access_token');
        const state = params.get('state');
        const savedState = sessionStorage.getItem('oauth_state');

        if (state && savedState && state !== savedState) {
            alert('State mismatch! Possible CSRF attack.');
            navigate('/login');
            return;
        }

        if (!token) {
            const error = params.get('error');
            if (error) {
                alert(`Authorization failed: ${error}`);
            }
            navigate('/login');
            return;
        }

        // Сохраняем токен
        sessionStorage.setItem('access_token', token);

        // Пытаемся извлечь роль из payload JWT
        try {
            const payloadBase64 = token.split('.')[1]
                .replace(/-/g, '+')
                .replace(/_/g, '/');
            const payloadJson = atob(payloadBase64);
            const payload = JSON.parse(payloadJson);
            const role = payload.role || 'User';
            sessionStorage.setItem('role', role);
        } catch (e) {
            console.warn('Failed to decode JWT payload', e);
            sessionStorage.setItem('role', 'User');
        }

        sessionStorage.removeItem('oauth_state');

        // Полный перезаход на главную, чтобы App перечитал sessionStorage
        window.location.replace('/');
    }, [navigate]);

    return <div className="loading">Loading...</div>;
}

export default Callback;