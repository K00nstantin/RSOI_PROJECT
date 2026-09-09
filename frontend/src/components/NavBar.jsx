import { Link, useNavigate } from 'react-router-dom';

function NavBar() {
    const token = sessionStorage.getItem('access_token');
    const role = sessionStorage.getItem('role');
    const navigate = useNavigate();

    const logout = () => {
        sessionStorage.removeItem('access_token');
        sessionStorage.removeItem('role');
        navigate('/login');
    };

    return (
        <nav className="navbar">
            <Link to="/">Libraries</Link>
            <Link to="/reservations">My Reservations</Link>
            <Link to="/rating">My Rating</Link>
            {role === 'Admin' && <Link to="/stats">Statistics Report</Link>}
            {role === 'Admin' && <Link to="/create-user">Create User</Link>}
            {token && <button onClick={logout} className="logout-btn">Logout</button>}
            {!token && <Link to="/login">Login</Link>}
        </nav>
    );
}

export default NavBar;