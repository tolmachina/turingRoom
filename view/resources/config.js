// config.js
const CONFIG = {
    development: {
        apiUrl: 'http://localhost:8080/api',
        wsUrl: 'ws://localhost:8080/ws'
    },
    production: {
        apiUrl: 'https://your-production-api.com/api',
        wsUrl: 'wss://your-production-api.com/ws'
    }
};

const ENV = 'development'; // Change this to 'production' when deploying

const getConfig = () => CONFIG[ENV];