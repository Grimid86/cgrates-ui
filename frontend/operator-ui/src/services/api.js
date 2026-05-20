import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
})

let isRefreshing = false
let refreshSubscribers = []

function onRefreshed(newToken) {
  refreshSubscribers.forEach((callback) => callback(newToken))
  refreshSubscribers = []
}

function addRefreshSubscriber(callback) {
  refreshSubscribers.push(callback)
}

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('operator_access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  const csrf = document.cookie.match(/csrf_token=([^;]+)/)?.[1]
  if (csrf) {
    config.headers['X-CSRF-Token'] = csrf
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && originalRequest && !originalRequest._retry) {
      if (originalRequest.url === '/auth/refresh') {
        localStorage.removeItem('operator_access_token')
        localStorage.removeItem('operator_refresh_token')
        localStorage.removeItem('operator_user')
        if (!window.location.pathname.includes('/login')) {
          window.location.href = '/login'
        }
        return Promise.reject(error)
      }

      originalRequest._retry = true

      if (!isRefreshing) {
        isRefreshing = true
        const refreshToken = localStorage.getItem('operator_refresh_token')
        if (!refreshToken) {
          isRefreshing = false
          localStorage.removeItem('operator_access_token')
          localStorage.removeItem('operator_refresh_token')
          localStorage.removeItem('operator_user')
          if (!window.location.pathname.includes('/login')) {
            window.location.href = '/login'
          }
          return Promise.reject(error)
        }

        try {
          const { data } = await axios.post(
            (import.meta.env.VITE_API_BASE_URL || '/api/v1') + '/auth/refresh',
            { refresh_token: refreshToken }
          )
          localStorage.setItem('operator_access_token', data.access_token)
          localStorage.setItem('operator_refresh_token', data.refresh_token)
          isRefreshing = false
          onRefreshed(data.access_token)
        } catch (refreshError) {
          isRefreshing = false
          refreshSubscribers = []
          localStorage.removeItem('operator_access_token')
          localStorage.removeItem('operator_refresh_token')
          localStorage.removeItem('operator_user')
          if (!window.location.pathname.includes('/login')) {
            window.location.href = '/login'
          }
          return Promise.reject(refreshError)
        }
      }

      return new Promise((resolve) => {
        addRefreshSubscriber((newToken) => {
          originalRequest.headers.Authorization = `Bearer ${newToken}`
          resolve(api(originalRequest))
        })
      })
    }

    return Promise.reject(error)
  }
)

export default api
