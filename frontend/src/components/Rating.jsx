import { useState, useEffect } from 'react';
import api from '../api/client';

function Rating() {
    const [stars, setStars] = useState(null);

    useEffect(() => {
        fetchRating();
    }, []);

    const fetchRating = async () => {
        try {
            const response = await api.get('/rating');
            setStars(response.data.stars);
        } catch (err) {
            console.error(err);
        }
    };

    return (
        <div>
            <h2>My Rating</h2>
            {stars !== null ? <p className="rating-stars">★ {stars}</p> : <p>Loading...</p>}
        </div>
    );
}

export default Rating;
