function LoginButton() {
    const startAuth = () => {
        const state = Math.random().toString(36).substring(7);
        sessionStorage.setItem('oauth_state', state);
        const clientId = 'SPA%20Application';
        const redirectUri = 'http://localhost:3000/callback';
        const scope = 'openid profile email';
        const url = `http://localhost:8090/api/v1/authorize?response_type=token&client_id=${clientId}&redirect_uri=${redirectUri}&scope=${scope}&state=${state}`;
        window.location.href = url;
    };

    return (
        <div className="login-container">
            <h2>Welcome to Library System</h2>
            <button onClick={startAuth} className="login-btn">Sign in with Identity Provider</button>
        </div>
    );
}

export default LoginButton;