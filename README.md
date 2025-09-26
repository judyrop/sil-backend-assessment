# SIL Backend Engineer Assessment

## Overview

This project is a Go-based backend service for a simplified e-commerce system. It manages customers, categories, products, and orders, with authentication via OpenID Connect. The service exposes a REST API and is containerized for deployment on Kubernetes clusters.

---

## Features

- **Database**: PostgreSQL (`sil` database)
- **API**: REST API for products, categories, customers, and orders
- **Authentication**: OpenID Connect for customers
- **Notifications**:
  - SMS alerts to customers via Africa’s Talking sandbox
  - Email notifications to administrators on order placement
- **Testing**:
  - Unit tests with coverage

- **Deployment**:
  - Dockerized service
  - Deployed on Kubernetes (Kind cluster used for local testing)
  - CI/CD pipelines for automated build, test, and deployment



## Prerequisites

- Docker
- Kubernetes cluster (Kind or Minikube)
- kubectl
- Go (for building locally)
- PostgreSQL client (optional for direct database queries)




