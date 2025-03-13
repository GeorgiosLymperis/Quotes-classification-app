import requests
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry
from bs4 import BeautifulSoup
import logging
import json
import pickle
from datetime import datetime

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
    handlers=[
        logging.FileHandler('scraper.log', mode="a", encoding="utf-8")])

class BaseScraper:
    def __init__(self):
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"
        })
        retries = Retry(total=5, backoff_factor=5, status_forcelist=[500, 502, 503, 504])
        self.session.mount("https://", HTTPAdapter(max_retries=retries))

    def get_page(self, url):
        try:
            page = self.session.get(url, timeout=30)
            page.raise_for_status()
            logging.info('Scrape page %s is finished', url)
            return page.text
        except requests.RequestException as e:
            logging.error("Error fetching page: %s", e)
            return ''
        
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