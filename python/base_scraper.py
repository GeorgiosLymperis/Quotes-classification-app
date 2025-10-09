import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry
from bs4 import BeautifulSoup
import logging
import json
import pickle
from datetime import datetime

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
    handlers=[
        logging.FileHandler('scraper.log', mode="a", encoding="utf-8")
    ]
)

class BaseScraper:
    """
    A base scraper class that provides essential web scraping functionalities.
    
    This class sets up a session with retry mechanisms, handles page requests,
    parsing, and provides methods for saving scraped data. It is meant to be 
    subclassed, requiring child classes to implement specific parsing methods.
    """
    
    def __init__(self):
        """
        Initializes the scraper session with a User-Agent header and retry mechanism.
        
        The retry mechanism will attempt failed requests up to 5 times with an increasing
        backoff factor of 5 seconds.
        """
        self.session = requests.Session()
        self.session.headers.update({
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"
        })
        retries = Retry(total=5, backoff_factor=5, status_forcelist=[400, 500, 502, 503, 504])
        self.session.mount("https://", HTTPAdapter(max_retries=retries))

    def get_page(self, url: str) -> str:
        """
        Fetches the HTML content of a given URL.
        
        Args:
            url (str): The URL of the webpage to be scraped.
        
        Returns:
            str: The HTML content of the page as a string, or an empty string if the request fails.
        """
        try:
            page = self.session.get(url, timeout=30)
            page.raise_for_status()
            logging.info('Scrape page %s is finished', url)
            return page.text
        except requests.RequestException as e:
            logging.error("Error fetching page: %s", e)
            return ''
        
    def parse_page(self, page: str) -> BeautifulSoup:
        """
        Parses the HTML content using BeautifulSoup.
        
        Args:
            page (str): The HTML content as a string.
        
        Returns:
            BeautifulSoup: A BeautifulSoup object representing the parsed HTML.
        """
        return BeautifulSoup(page, 'html.parser')
    
    def get_quote(self, block):
        """
        Extracts the quote text from a given HTML block.
        
        This method must be implemented in a subclass.
        
        Args:
            block (BeautifulSoup element): A BeautifulSoup element representing a quote block.
        
        Raises:
            NotImplementedError: If the method is not implemented in a subclass.
        """
        raise NotImplementedError("get_quote method must be implemented")
    
    def get_author(self, block):
        """
        Extracts the author of a quote from a given HTML block.
        
        This method must be implemented in a subclass.
        
        Args:
            block (BeautifulSoup element): A BeautifulSoup element containing author information.
        
        Raises:
            NotImplementedError: If the method is not implemented in a subclass.
        """
        raise NotImplementedError("get_author method must be implemented")
    
    def get_tags(self, block):
        """
        Extracts tags or categories related to a quote from a given HTML block.
        
        This method must be implemented in a subclass.
        
        Args:
            block (BeautifulSoup element): A BeautifulSoup element containing tags.
        
        Raises:
            NotImplementedError: If the method is not implemented in a subclass.
        """
        raise NotImplementedError("get_tags method must be implemented")
    
    def get_likes(self, block):
        """
        Extracts the number of likes or interactions from a given HTML block.
        
        This method must be implemented in a subclass.
        
        Args:
            block (BeautifulSoup element): A BeautifulSoup element containing like information.
        
        Raises:
            NotImplementedError: If the method is not implemented in a subclass.
        """
        raise NotImplementedError("get_likes method must be implemented")
    
    def scrape_page(self, url: str):
        """
        Scrapes a single webpage for quotes and associated metadata.
        
        This method must be implemented in a subclass.
        
        Args:
            url (str): The URL of the webpage to scrape.
        
        Raises:
            NotImplementedError: If the method is not implemented in a subclass.
        """
        raise NotImplementedError("scrape method must be implemented")
    
    def _save(self, data: list, filename: str):
        """
        Saves scraped data to a file in either JSON or pickle format.
        
        The filename is automatically suffixed with a timestamp.
        
        Args:
            data (list): The scraped data to be saved.
            filename (str): The base filename (with `.json` or `.pkl` extension).
        
        Raises:
            NotImplementedError: If the file format is not JSON or pickle.
        """
        now = datetime.now().strftime("%m_%d_%Y_%H%M%S")
        filename_split = filename.split('.')
        file, ext = '.'.join(filename_split[:-1]), filename_split[-1]
        filename = f'{file}_{now}.{ext}'
        
        if 'json' in filename:
            with open(filename, 'w', encoding='utf-8') as f:
                json.dump(data, f, ensure_ascii=False, indent=4)

        elif 'pkl' in filename:
            with open(filename, 'wb') as f:
                pickle.dump(data, f)

        else:
            raise NotImplementedError("Only json and pickle supported")
