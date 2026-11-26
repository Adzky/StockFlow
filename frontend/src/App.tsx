import { Link, Route, Routes } from 'react-router-dom'
import { ProductsPage } from './pages/ProductsPage'
import { CreateOrderPage } from './pages/CreateOrderPage'
import { OrdersPage } from './pages/OrdersPage'
import { OrderDetailPage } from './pages/OrderDetailPage'

export default function App() {
  return (
    <>
      <nav>
        <Link to="/">Products</Link>
        {' | '}
        <Link to="/orders">Orders</Link>
        {' | '}
        <Link to="/orders/new">New order</Link>
      </nav>

      <Routes>
        <Route path="/" element={<ProductsPage />} />
        <Route path="/orders" element={<OrdersPage />} />
        <Route path="/orders/new" element={<CreateOrderPage />} />
        <Route path="/orders/:id" element={<OrderDetailPage />} />
      </Routes>
    </>
  )
}