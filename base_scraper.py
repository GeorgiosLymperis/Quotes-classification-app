import requests
from bs4 import BeautifulSoup
import logging
import json
import pickle
from datetime import datetime

logging.basicConfig(level=logging.INFO)

class BaseScraper:
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": "Personal Project"
        })

    def get_page(self, url):
        try:
            page = self.session.get(url, timeout=10)
            page.raise_for_status()
            return page.text
        except requests.RequestException as e:
            logging.error("Error fetching page: %s", e)
            return None
        
    def parse_page(self, page):
        soup = BeautifulSoup(page, 'html.parser')
        return soup
    
    def get_quote(self, block):
        raise NotImplementedError("get_quote method must be implemented")
    
    def get_author(self, block):
        raise NotImplementedError("get_author method must be implemented")
    
    def get_tags(self, block):
        raise NotImplementedError("get_tags method must be implemented")
    
    def get_likes(self, block):
        raise NotImplementedError("get_likes method must be implemented")
    
    def scrape_page(self, url):
        raise NotImplementedError("scrape method must be implemented")
    
    def _save(self, data: list, filename: str):
        now = datetime.now().strftime("%m_%d_%Y_%H%M%S")
        filename_split = filename.split('.')
        file, ext = '.'.join(filename_split[:-1]), filename_split[-1]
        filename = f'{file}_{now}.{ext}'
        if 'json' in filename:
            with open(filename, 'w') as f:
                json.dump(data, f)

        elif 'pkl' in filename:
            with open(filename, 'wb') as f:
                pickle.dump(data, f)

        else:
            raise NotImplementedError("Only json and pickle supported")