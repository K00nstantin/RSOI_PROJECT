import { useState, useEffect } from 'react';
import api from '../api/client';

function Statistics() {
    const [stats, setStats] = useState(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    useEffect(() => {
        const token = sessionStorage.getItem('access_token');
        if (!token) {
            setLoading(false);
            return;
        }
        const fetchStats = async () => {
            try {
                const response = await api.get('/stats/report');
                setStats(response.data);
            } catch (err) {
                setError(err.message);
            } finally {
                setLoading(false);
            }
        };
        fetchStats();
    }, []);

    if (loading) return <div>Loading...</div>;
    if (error) return <div>Error: {error}</div>;
    if (!stats) return <div>No data</div>;

    return (
        <div>
            <h2>Statistics Report</h2>
            <p><strong>Total events:</strong> {stats.total_events}</p>
            <h3>Breakdown by action</h3>
            <ul>
                {Object.entries(stats.breakdown).map(([action, count]) => (
                    <li key={action}>{action}: {count}</li>
                ))}
            </ul>
        </div>
    );
}

export default Statistics;