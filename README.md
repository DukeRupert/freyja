# ☕ Freyja - Coffee E-commerce Platform

> Modern, event-driven e-commerce platform designed specifically for coffee roasting businesses

[![Go Version](https://img.shields.io/github/go-mod/go-version/dukerupert/freyja)](https://golang.org/)
[![License](https://img.shields.io/github/license/dukerupert/freyja)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue)](docker-compose.yml)
[![Monitoring](https://img.shields.io/badge/monitoring-prometheus-orange)](config/prometheus/)

## 🚀 Quick Start

```bash
# Clone and setup
git clone https://github.com/dukerupert/freyja.git
cd freyja
make setup

# Start infrastructure
make start

# Access services
open http://localhost:8080    # Application
open http://localhost:3000    # Grafana (admin/grafana_admin_123)
✨ Features

🛒 Retail E-commerce - Complete online store with cart and checkout
📦 Subscription Management - Flexible recurring orders and billing
🏢 B2B Wholesale - Tiered pricing and NET-30 terms
📊 Business Analytics - Real-time metrics and insights
⚡ Event-Driven - Reliable workflows with NATS JetStream
🔒 Security First - JWT auth, RBAC, audit trails
📈 Observability - Prometheus metrics and Grafana dashboards

🏗️ Architecture
Built on modern, cloud-native principles:

Backend: Go + Echo + PostgreSQL + SQLC
Caching: Valkey (Redis fork)
Events: NATS JetStream
Storage: MinIO/S3-compatible
Monitoring: Prometheus + Grafana + AlertManager
Deployment: Docker Compose → Kubernetes

📚 Documentation

Architecture Decision Records - Key technical decisions
Getting Started - Setup and development guide
API Documentation - RESTful API reference
Deployment Guide - Production setup

🤝 Contributing
We welcome contributions! See CONTRIBUTING.md for guidelines.
📄 License
MIT License - see LICENSE file for details.
