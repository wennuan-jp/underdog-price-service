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
- [ ] Create a new project in the [Firebase Console](https://console.firebase.google.com/).
- [ ] Choose **Cloud Firestore** and create a database in your preferred region.
- [ ] Set security rules (locked for now, as we use Admin SDK).

### 2. Service Account Authentication
- [ ] Go to Project Settings > Service Accounts.
- [ ] Generate a new private key and save the JSON file as `_firebase_credentials.json` (ensure this is in `.gitignore`).
- [ ] Set environment variable `GOOGLE_APPLICATION_CREDENTIALS` pointing to this file.

### 3. Initialize Firebase Admin SDK
- [ ] Install the Go SDK:
  ```bash
  go get firebase.google.com/go/v4
  ```
- [ ] Create `infra/firebase.go` to initialize the Firebase App and Firestore client.

### 4. Data Mapping & Structure
- [ ] Decide on the collection structure (e.g., `prices` collection with asset IDs as document IDs).
- [ ] Map `PricePair` struct to Firestore tags:
  ```go
  type PricePair struct {
      ID          string    `firestore:"id"`
      Name        string    `firestore:"name"`
      AssetType   AssetType `firestore:"asset_type"`
      Code        string    `firestore:"code"`
      PriceInUSD  float64   `firestore:"price_usd"`
      LastUpdated time.Time `firestore:"last_updated"`
  }
  ```

### 5. Implement Price Pushing Service
- [ ] Create `service/firebase_service.go`.
- [ ] Implement `UpdatePrice(ctx context.Context, price PricePair)` using `Set` with merge options.

### 6. Integration in Main Loop
- [ ] Update `main.go` to initialize Firebase on startup.
- [ ] After fetching new prices, call `FirebaseService.UpdatePrice` for each asset.

### 7. Verification
- [ ] Monitor the Firebase Console to see real-time updates.
- [ ] (Optional) Create a simple web/mobile client with a Firestore listener to verify the "automatic push" behavior.


