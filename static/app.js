const API_URL = '/api/products';
const AUTH_URL = '/api/login';
let currentToken = '';

// Функция для показа/скрытия элементов в зависимости от роли
function updateUIForRole(role) {
    const adminElements = document.querySelectorAll('.admin-only');
    const userElements = document.querySelectorAll('.user-only');
    
    adminElements.forEach(el => {
        el.style.display = role === 'admin' ? 'block' : 'none';
    });
    
    userElements.forEach(el => {
        el.style.display = role === 'user' ? 'block' : 'none';
    });
}

// Логин
async function login() {
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;

    try {
        const response = await fetch(AUTH_URL, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password })
        });

        if (response.ok) {
            const data = await response.json();
            currentToken = data.token;
            
            document.getElementById('loginSection').style.display = 'none';
            document.getElementById('mainSection').style.display = 'block';
            document.getElementById('userInfo').textContent = 
                `Logged in as: ${data.user} (${data.role})`;
            
            updateUIForRole(data.role);
            loadProducts();
        } else {
            alert('Login failed');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Login error');
    }
}

// Выход
function logout() {
    currentToken = '';
    document.getElementById('loginSection').style.display = 'block';
    document.getElementById('mainSection').style.display = 'none';
    document.getElementById('products').innerHTML = '';
}

// Загрузка продуктов с авторизацией
async function loadProducts() {
    try {
        const headers = {};
        if (currentToken) {
            headers['Authorization'] = `Bearer ${currentToken}`;
        }

        const response = await fetch(API_URL, { headers });
        const products = await response.json();
        displayProducts(products);
    } catch (error) {
        console.error('Error loading products:', error);
    }
}

// Остальные функции (displayProducts, searchByCategory, etc.) остаются похожими,
// но добавляем проверку авторизации для защищенных endpoints

async function addProduct(product) {
    const response = await fetch(API_URL, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${currentToken}`
        },
        body: JSON.stringify(product)
    });

    if (response.status === 403) {
        alert('Access denied: Admin role required');
        return false;
    }
    
    return response.ok;
}