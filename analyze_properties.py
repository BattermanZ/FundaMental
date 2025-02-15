from funda.data.database import FundaDB
import pandas as pd
from datetime import datetime

def analyze_properties():
    db = FundaDB()
    
    # Get basic stats
    stats = db.get_basic_stats()
    print("\n=== Basic Statistics ===")
    print(f"Total properties: {stats['total_properties']}")
    print(f"Average price: €{stats['avg_price']:,.2f}")
    print(f"Average days to sell: {stats['avg_days_to_sell']:.1f} days")
    
    # Get recent sales
    recent_sales = db.get_recent_sales(limit=5)
    print("\n=== Recent Sales ===")
    for sale in recent_sales:
        print(f"{sale['street']} - €{sale['price']:,} - Sold on: {sale['selling_date']}")
    
    # Analyze by postal code (first 4 digits)
    print("\n=== Analysis by Area (Postal Code) ===")
    for postal_prefix in ['1011', '1012', '1013']:  # Add more postal codes as needed
        properties = db.get_properties_by_postal_code(postal_prefix)
        if properties:
            prices = [p['price'] for p in properties if p['price']]
            avg_price = sum(prices) / len(prices) if prices else 0
            print(f"\nPostal code {postal_prefix}:")
            print(f"Number of properties: {len(properties)}")
            print(f"Average price: €{avg_price:,.2f}")

if __name__ == "__main__":
    analyze_properties() 