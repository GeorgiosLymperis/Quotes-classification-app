from base_scraper import BaseScraper
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Dict

class GoodReadsScraper(BaseScraper):
    """
    A web scraper for Goodreads that extracts quotes, topics, and authors from the site.

    This scraper supports:
    - Scraping individual pages for quotes
    - Extracting topics and associated URLs
    - Multi-threaded scraping for multiple pages concurrently
    """
    def scrape_page(self, url: str, save: bool = False)->List[Dict[str, str]]:
        """
        Scrapes a single page for quotes, extracting relevant details such as the quote, author, tags, and likes.

        Args:
            url (str): The URL of the page to scrape.
            save (bool): If True, saves the extracted quotes to a pickle file.

        Returns:
            List[Dict[str, str]]: A list of dictionaries containing the quote, author, tags, and likes.
        """
        page = self.get_page(url)
        soup = self.parse_page(page)
        quotes = []

        for block in soup.find_all('div', class_='quote mediumText'):
            quotes.append({
                'quote': self.get_quote(block),
                'author': self.get_author(block),
                'tags': self.get_tags(block),
                'likes': self.get_likes(block)
            })

        if save == True:
            self._save(quotes, 'quotes_qr.pkl')
        return quotes
    @staticmethod
    def get_quote(block)->str:
        """
        Extracts the quote text from a quote block.

        Args:
            block (BeautifulSoup element): The block containing the quote.

        Returns:
            str: The extracted quote text.
        """
        return block.find('div', class_='quoteText').get_text(strip=True)
    
    @staticmethod
    def get_author(block)->str:
        """
        Extracts the author of the quote.

        Args:
            block (BeautifulSoup element): The block containing the author.

        Returns:
            str: The extracted author name, or 'Unknown' if not found.
        """
        author_tag = block.find('span', class_='authorOrTitle').get_text(strip=True)
        return author_tag if author_tag else 'Unknown'

    @staticmethod
    def get_likes(block)->int:
        """
        Extracts the number of likes for a quote.

        Args:
            block (BeautifulSoup element): The block containing like information.

        Returns:
            int: The number of likes as an integer.
        """
        likes_box = block.find('div', class_='quoteFooter')
        likes = likes_box.find('a', class_='smallText').get_text(strip=True)
        return likes.split(' ')[0]
    
    @staticmethod
    def get_tags(block)->List[str]:
        """
        Extracts the tags associated with a quote.

        Args:
            block (BeautifulSoup element): The block containing tags.

        Returns:
            List[str]: A list of tag strings.
        """
        tags = []
        tags_block = block.find('div', class_='quoteFooter')
        for tag in tags_block.find_all('a'):
            tags.append(tag.get_text())
        return tags[:-1]
    
    def scrape_many_pages(self, urls: List[str], start_page: int, end_page: int, save: bool = False)->List[Dict[str, str]]:
        """
        Scrapes multiple pages concurrently and extracts quotes.

        Args:
            urls (List[str]): A list of URLs to scrape.
            start_page (int): The starting page number for pagination.
            end_page (int): The last page number to scrape.
            save (bool): If True, saves the extracted data to a pickle file.

        Returns:
            List[Dict[str, str]]: A list of dictionaries containing quote data from all pages.
        """
        quotes = []
        futures = []
        with ThreadPoolExecutor(max_workers=10) as executor:
            for url in urls:
                for page in self.__generate_pages(url, start_page, end_page):
                    future = executor.submit(self.scrape_page, page)
                    futures.append(future)

            for future in as_completed(futures):
                quotes.extend(future.result())

        if save == True:
            self._save(quotes, 'quotes_qr.pkl')

        return quotes
    
    def scrape_topics(self, save: bool = False)->List[Dict[str, str]]:
        """
        Scrapes all available topics from Goodreads and their URLs.

        Args:
            save (bool): If True, saves the extracted topics to a pickle file.

        Returns:
            List[Dict[str, str]]: A list of dictionaries containing topic names and URLs.
        """
        url = 'https://www.goodreads.com/quotes'
        response = self.get_page(url)
        soup = self.parse_page(response)
        topics = []
        for topic in soup.find_all('li', class_='greyText'):
            topic_of_quote = topic.get_text(strip=True).split(' ')[0]
            topic_url = topic.find('a').get('href')
            topic_url = 'https://www.goodreads.com' + topic_url
            topics.append({'topic': topic_of_quote,'url': topic_url})

        if save == True:
            self._save(topics, 'topics_qr.pkl')
        return topics
    
    def __generate_pages(self, url: str, start_page: int, end_page: int):
        """
        Generates a list of paginated URLs for a given base URL.

        Args:
            url (str): The base URL for pagination.
            start_page (int): The starting page number.
            end_page (int): The last page number.

        Returns:
            List[str]: A list of fully constructed paginated URLs.
        """
        base_url = f"{url}?page="
        for i in range(start_page, end_page + 1):
            yield base_url + str(i)
