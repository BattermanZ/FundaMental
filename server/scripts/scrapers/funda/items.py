# -*- coding: utf-8 -*-

from dataclasses import dataclass
from typing import Optional
from datetime import datetime

@dataclass
class FundaItem:
    """Data structure for a Funda property listing."""
    url: str
    street: Optional[str] = None
    neighborhood: Optional[str] = None
    property_type: Optional[str] = None
    city: Optional[str] = None
    postal_code: Optional[str] = None
    price: Optional[int] = None
    year_built: Optional[int] = None
    living_area: Optional[int] = None
    num_rooms: Optional[int] = None
    status: Optional[str] = None
    listing_date: Optional[str] = None
    selling_date: Optional[str] = None
    scraped_at: str = datetime.now().isoformat()

    def to_dict(self) -> dict:
        """Convert item to dictionary for JSON serialization."""
        return {k: v for k, v in self.__dict__.items() if v is not None} 