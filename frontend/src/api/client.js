import axios from 'axios';

const api = axios.create({
    baseURL: process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1',
});

api.interceptors.request.use(config => {
    const token = sessionStorage.getItem('access_token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

api.interceptors.response.use(
    response => response,
    error => {
        // Не редиректим автоматически и не удаляем токен —
        // пусть компонент сам решает, что делать с 401.
        return Promise.reject(error);
    }
);

export default api;