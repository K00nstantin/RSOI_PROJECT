import { useState } from 'react';
import api from '../api/client';

function CreateUser() {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [email, setEmail] = useState('');
    const [role, setRole] = useState('User');
    const [message, setMessage] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        try {
            await api.post('/createUser', { username, password, email, role });
            setMessage('User created successfully!');
            setUsername(''); setPassword(''); setEmail(''); setRole('User');
        } catch (err) {
            if (err.response?.status === 403) {
                setMessage('Forbidden: Admin role required.');
            } else {
                setMessage('Failed to create user.');
            }
        }
    };

    return (
        <div>
            <h2>Create User (Admin only)</h2>
            <form onSubmit={handleSubmit} className="create-user-form">
                <input type="text" placeholder="Username" value={username} onChange={(e) => setUsername(e.target.value)} required />
                <input type="password" placeholder="Password" value={password} onChange={(e) => setPassword(e.target.value)} required />
                <input type="email" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} />
                <select value={role} onChange={(e) => setRole(e.target.value)}>
                    <option value="User">User</option>
                    <option value="Admin">Admin</option>
                </select>
                <button type="submit">Create</button>
            </form>
            {message && <p className="message">{message}</p>}
        </div>
    );
}

export default CreateUser;