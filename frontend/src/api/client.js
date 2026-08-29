import axios from 'axios';

const api = axios.create({
    baseURL: 'http://localhost:8080/api/v1',
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
        if (error.response?.status === 401) {
            sessionStorage.removeItem('access_token');
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);

export default api;