import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../api/client';

function ReturnBook() {
    const { reservationUid } = useParams();
    const navigate = useNavigate();
    const [condition, setCondition] = useState('EXCELLENT');
    const [date, setDate] = useState('');

    const handleReturn = async () => {
        if (!date) {
            alert('Please select return date');
            return;
        }
        try {
            await api.post(`/reservations/${reservationUid}/return`, { condition, date });
            alert('Book returned successfully!');
            navigate('/reservations');
        } catch (err) {
            alert('Failed to return book.');
        }
    };

    return (
        <div>
            <h2>Return Book</h2>
            <div className="form-group">
                <label>Condition:</label>
                <select value={condition} onChange={(e) => setCondition(e.target.value)}>
                    <option value="EXCELLENT">Excellent</option>
                    <option value="GOOD">Good</option>
                    <option value="BAD">Bad</option>
                </select>
            </div>
            <div className="form-group">
                <label>Return Date:</label>
                <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </div>
            <button onClick={handleReturn}>Confirm Return</button>
        </div>
    );
}

export default ReturnBook;