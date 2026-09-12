import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import NavBar from './components/NavBar';
import Libraries from './components/Libraries';
import Books from './components/Books';
import Reservations from './components/Reservations';
import ReturnBook from './components/ReturnBook';
import Rating from './components/Rating';
import CreateUser from './components/CreateUser';
import Callback from './components/Callback';
import LoginButton from './components/LoginButton';
import Statistics from './components/Statistics';
import './App.css';

function App() {
    const [token, setToken] = useState(sessionStorage.getItem('access_token'));

    useEffect(() => {
        // Перечитываем токен при каждом изменении URL
        const handler = () => setToken(sessionStorage.getItem('access_token'));
        window.addEventListener('popstate', handler);
        // Также проверяем токен сразу после монтирования
        handler();
        return () => window.removeEventListener('popstate', handler);
    }, []);

    return (
        <BrowserRouter>
            <NavBar />
            <div className="container">
                <Routes>
                    <Route path="/callback" element={<Callback />} />
                    <Route path="/login" element={<LoginButton />} />
                    {!token ? (
                        <Route path="*" element={<Navigate to="/login" />} />
                    ) : (
                        <>
                            <Route path="/" element={<Libraries />} />
                            <Route path="/libraries/:libraryUid/books" element={<Books />} />
                            <Route path="/reservations" element={<Reservations />} />
                            <Route path="/return/:reservationUid" element={<ReturnBook />} />
                            <Route path="/rating" element={<Rating />} />
                            <Route path="/create-user" element={<CreateUser />} />
                            <Route path="/stats" element={<Statistics />} />
                            <Route path="*" element={<Navigate to="/" />} />
                        </>
                    )}
                </Routes>
            </div>
        </BrowserRouter>
    );
}

export default App;