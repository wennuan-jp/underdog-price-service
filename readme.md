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

### 1. Setup Firebase Project
- [x] Create a new project in the [Firebase Console](https://console.firebase.google.com/).
- [x] Choose **Cloud Firestore** and create a database in your preferred region.
- [x] Set security rules (locked for now, as we use Admin SDK).

### 2. Service Account Authentication
- [x] Go to Project Settings > Service Accounts.
- [x] Generate a new private key and save the JSON file as `_firebase_credentials.json` (ensure this is in `.gitignore`).
- [x] Initialized by pointing to this file in `infra/firebase.go`.

### 3. Initialize Firebase Admin SDK
- [x] Install the Go SDK.
- [x] Create `infra/firebase.go` to initialize the Firebase App and Firestore client.


### 4. Data Mapping & Structure
- [x] Decide on the collection structure: Use a `prices` collection where each document ID is the asset `Code` (e.g., `USD`, `BTC`).
- [x] Map `PricePair` struct to Firestore tags in `model/model.go`.

### 5. Implement Price Pushing Service
- [x] Create `service/firebase_service.go`.
- [x] Implement `UpdatePrice(ctx context.Context, price model.PricePair)` using `Set` with merge options.



### 6. Integration in Main Loop
- [x] Update `main.go` to initialize Firebase on startup.
- [x] After fetching new prices, call `FirebaseService.UpdatePrice` for each asset.


### 7. Verification
- [ ] Monitor the Firebase Console to see real-time updates.
- [x] (Optional) Create a simple web/mobile client with a Firestore listener to verify the "automatic push" behavior.

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




