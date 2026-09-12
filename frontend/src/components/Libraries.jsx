import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../api/client';

function Libraries() {
    const [libraries, setLibraries] = useState([]);
    const [city, setCity] = useState('Москва');
    const [page, setPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const navigate = useNavigate();

    useEffect(() => {
        const token = sessionStorage.getItem('access_token');
        if (!token) {
            return;
        }
        fetchLibraries();
    }, [city, page]);

    const fetchLibraries = async () => {
        try {
            const response = await api.get(`/libraries?page=${page}&size=10&city=${city}`);
            setLibraries(response.data.items || []);
            setTotalPages(Math.ceil((response.data.totalElements || 0) / 10));
        } catch (err) {
            console.error(err);
        }
    };

    return (
        <div>
            <h2>Libraries</h2>
            <div className="search">
                <input
                    type="text"
                    value={city}
                    onChange={(e) => setCity(e.target.value)}
                    placeholder="City"
                />
                <button onClick={fetchLibraries}>Search</button>
            </div>
            <ul className="library-list">
                {libraries.map(lib => (
                    <li key={lib.libraryUid}>
                        <h3>{lib.name}</h3>
                        <p>{lib.address}, {lib.city}</p>
                        <button onClick={() => navigate(`/libraries/${lib.libraryUid}/books`)}>
                            View Books
                        </button>
                    </li>
                ))}
            </ul>
            <div className="pagination">
                <button disabled={page === 1} onClick={() => setPage(p => p - 1)}>Prev</button>
                <span>Page {page} of {totalPages}</span>
                <button disabled={page === totalPages} onClick={() => setPage(p => p + 1)}>Next</button>
            </div>
        </div>
    );
}

export default Libraries;