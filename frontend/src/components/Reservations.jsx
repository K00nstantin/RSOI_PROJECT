import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../api/client';

function Reservations() {
    const [reservations, setReservations] = useState([]);
    const navigate = useNavigate();

    useEffect(() => {
        fetchReservations();
    }, []);

    const fetchReservations = async () => {
        try {
            const response = await api.get('/reservations');
            setReservations(response.data || []);
        } catch (err) {
            console.error(err);
        }
    };

    return (
        <div>
            <h2>My Reservations</h2>
            <ul className="reservation-list">
                {reservations.map(res => (
                    <li key={res.reservationUid}>
                        <h3>{res.book.name}</h3>
                        <p>Status: {res.status}</p>
                        <p>From: {res.startDate} To: {res.tillDate}</p>
                        <p>Library: {res.library.name}</p>
                        {res.status === 'RENTED' && (
                            <button onClick={() => navigate(`/return/${res.reservationUid}`)}>Return</button>
                        )}
                    </li>
                ))}
            </ul>
        </div>
    );
}

export default Reservations;