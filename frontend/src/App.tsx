import { Link, Route, Routes } from 'react-router-dom'

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
        <Route path="/" element={<h1>Products</h1>} />
        <Route path="/orders" element={<h1>Orders</h1>} />
        <Route path="/orders/new" element={<h1>New order</h1>} />
        <Route path="/orders/:id" element={<h1>Order detail</h1>} />
      </Routes>
    </>
  )
}