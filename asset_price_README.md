This project is dedicated to fetch all the assets price and store it into Firebase, so that when data in Firebase updated, Firebase will automatically pushes the latest data to every connected client.

For example, this service will grab price of BTC, ETH, and CNY, etc. From different resources. Due to these resources are replaceable and consequently changed, this server must be maintainable and adaptive, so that when changes the resources, it will not need to change so many codes.

# Tasks
- [X] define the data structure
- [X] test out first price API
- [ ] define the structure of the data in Firebase
- [ ] push price data to firebase
- [ ] check that auto push mechanism is works

## Firebase Implementation Plan

To achieve real-time price updates for clients, we will use **Firebase Cloud Firestore**. Below is the step-by-step plan:


### 7. Verification
- [ ] Monitor the Firebase Console to see real-time updates.


## Usage (HTTP API)
The service now runs as an HTTP server on port `8080`.

### Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/health` | Check service status |
| `POST` | `/v1/admin/refresh-supported` | Refresh supported currency names list |
| `POST` | `/v1/price/sync-fx` | Fetch live FX rates and sync with Firebase |

### Example Manual Sync

**Sync FX Rates:**
```bash
curl -X POST http://localhost:8080/v1/price/sync-fx
```

**Refresh Currency Names:**
```bash
curl -X POST http://localhost:8080/v1/admin/refresh-supported
```




