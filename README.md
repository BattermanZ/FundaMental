# 🏠 FundaMental

A comprehensive real estate analysis platform for the Dutch property market, powered by data from Funda.nl.

## 🌟 Features

- 🗺️ Interactive property maps with price heatmaps
- 📊 Real-time market statistics and trends
- 🏘️ Metropolitan area analysis
- 📱 Telegram notifications for new listings
- 🔄 Automated data collection
- 📈 Historical price tracking
- 🎯 District-based analysis

## 🏗️ Architecture

The application is built with a modern tech stack:

### Frontend (client/)
- React with TypeScript
- Material-UI components
- Interactive maps (Leaflet)
- Data visualization (D3, Recharts)
- Real-time updates

### Backend (server/)
- Go server
- Python scrapers
- SQLite database
- Geocoding service
- Telegram integration

## 📁 Project Structure

```
fundamental/
├── client/                 # Frontend React application
│   ├── src/               # Source code
│   │   ├── components/    # React components
│   │   ├── services/      # API services
│   │   └── App.tsx        # Main application
│   └── public/            # Static assets
├── server/                # Backend Go application
│   ├── cmd/              # Entry points
│   ├── internal/         # Core backend logic
│   ├── config/           # Configuration
│   ├── database/         # SQLite database
│   └── scripts/          # Python scrapers
│       └── scrapers/     # Funda.nl scrapers
└── documentation/        # Project documentation
```

## 🚀 Getting Started

### Prerequisites
- Node.js 18+
- Go 1.24+
- Python 3.13+
- Docker (optional)

### Development Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd fundamental
```

2. Start the backend:
```bash
cd server
go mod download
go run cmd/server/main.go
```

3. Start the frontend:
```bash
cd client
npm install
npm start
```

### Docker Setup

Use Docker Compose to run the entire stack:
```bash
docker-compose up --build
```

## 🔧 Configuration

### Metropolitan Areas
Configure metropolitan areas in the UI or via the configuration file:
```json
{
    "metropolitan_areas": [
        {
            "name": "Amsterdam Metro",
            "cities": ["Amsterdam", "Amstelveen", "Diemen"]
        }
    ]
}
```

### Telegram Notifications
Set up Telegram notifications through the configuration interface for:
- New listings
- Price changes
- Market updates

## 🔄 Data Collection

The application uses two types of scrapers:
1. Active listings scraper (`funda_spider`)
2. Sold properties scraper (`funda_spider_sold`)

Data is collected and processed automatically with:
- Scheduled updates
- Real-time geocoding
- District boundary generation
- Price trend analysis

## 📊 Analytics Features

- Property price heatmaps
- Historical price trends
- District-based analysis
- Market statistics
- Metropolitan area comparisons
- Sale duration analysis

## 🌐 API Endpoints

The backend provides RESTful APIs for:
- Property data
- Metropolitan areas
- Market statistics
- Telegram configuration
- Spider management

## 🛠️ Development

### Frontend Development
```bash
cd client
npm install
npm start
```

### Backend Development
```bash
cd server
go mod download
go run cmd/server/main.go
```

### Database Migrations
Migrations run automatically on server start.

## 🔍 Monitoring

The application includes:
- Detailed logging
- Spider status tracking
- Geocoding progress
- Database statistics

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## 📝 Notes

- Rate limiting is implemented for API requests
- Geocoding is rate-limited
- Data is cached for performance
- District boundaries are auto-generated
