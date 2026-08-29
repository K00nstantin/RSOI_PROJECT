import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../api/client';

function Books() {
    const { libraryUid } = useParams();
    const [books, setBooks] = useState([]);
    const [page, setPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [showAll, setShowAll] = useState(false);
    const navigate = useNavigate();

    useEffect(() => {
        fetchBooks();
    }, [libraryUid, page, showAll]);

    const fetchBooks = async () => {
        try {
            const response = await api.get(`/libraries/${libraryUid}/books?page=${page}&size=10&showAll=${showAll}`);
            setBooks(response.data.items || []);
            setTotalPages(Math.ceil((response.data.totalElements || 0) / 10));
        } catch (err) {
            console.error(err);
        }
    };

    const takeBook = async (bookUid, tillDate) => {
        try {
            await api.post('/reservations', {
                bookUid,
                libraryUid,
                tillDate
            });
            alert('Book rented successfully!');
            fetchBooks();
        } catch (err) {
            if (err.response?.status === 403) {
                alert('Too many books rented or rating too low.');
            } else {
                alert('Failed to rent book.');
            }
        }
    };

    return (
        <div>
            <h2>Books in Library</h2>
            <div className="filters">
                <label>
                    <input type="checkbox" checked={showAll} onChange={() => setShowAll(!showAll)} />
                    Show all (including unavailable)
                </label>
            </div>
            <ul className="book-list">
                {books.map(book => (
                    <li key={book.bookUid}>
                        <h3>{book.name}</h3>
                        <p>Author: {book.author}</p>
                        <p>Genre: {book.genre}</p>
                        <p>Condition: {book.condition}</p>
                        <p>Available: {book.availableCount}</p>
                        {book.availableCount > 0 && (
                            <button onClick={() => {
                                const tillDate = prompt('Enter return date (YYYY-MM-DD):', '2026-09-01');
                                if (tillDate) takeBook(book.bookUid, tillDate);
                            }}>Rent</button>
                        )}
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

export default Books;