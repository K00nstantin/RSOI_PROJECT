import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

function Callback() {
    const navigate = useNavigate();

    useEffect(() => {
        const hash = window.location.hash;
        if (hash) {
            const params = new URLSearchParams(hash.slice(1));
            const token = params.get('access_token');
            const state = params.get('state');
            const savedState = sessionStorage.getItem('oauth_state');
            if (state && savedState && state !== savedState) {
                alert('State mismatch! Possible CSRF attack.');
                navigate('/login');
                return;
            }
            if (token) {
                sessionStorage.setItem('access_token', token);
                sessionStorage.removeItem('oauth_state');
                navigate('/');
            } else {
                const error = params.get('error');
                if (error) {
                    alert(`Authorization failed: ${error}`);
                }
                navigate('/login');
            }
        } else {
            navigate('/login');
        }
    }, [navigate]);

    return <div className="loading">Loading...</div>;
}

export default Callback;