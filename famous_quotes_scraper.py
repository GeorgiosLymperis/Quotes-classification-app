from base_scraper import BaseScraper
from urllib.parse import urljoin
from typing import List, Dict
from concurrent.futures import ThreadPoolExecutor, as_completed

class FamousQuotesScraper(BaseScraper):
    """
    A web scraper for extracting quotes, topics, and authors from FamousQuotesAndAuthors.com.

    This scraper supports:
    - Scraping individual pages for quotes
    - Extracting topics and associated URLs
    - Scraping author names and their respective URLs
    - Multi-threaded scraping for efficiency
    """

    def scrape_page(self, url: str, save: bool = False, savepath: str = 'quotes_fq.pkl') -> List[Dict[str, str]]:
        """
        Scrapes a single page for quotes, extracting relevant details such as the quote, author, tags, and likes.

        Args:
            url (str): The URL of the page to scrape.
            save (bool): If True, saves the extracted quotes to a pickle file.
            savepath (str): The path to save the extracted quotes. Must be a .pkl or .json file

        Returns:
            List[Dict[str, str]]: A list of dictionaries containing the quote, author, tags, and likes.
        """
        page = self.get_page(url)
        soup = self.parse_page(page)
        table = soup.find('td', style='padding-left:16px; padding-right:16px;', valign='top')

        quotes = self.get_quote(table)
        authors = self.get_author(table)
        likes = None
        tag = self.get_tags(table)

        assert len(quotes) == len(authors), "Mismatch between quotes and authors count."

        extracted_quotes = [
            {'quote': quote, 'author': author, 'tags': tag, 'likes': likes}
            for quote, author in zip(quotes, authors)
        ]

        if save:
            self._save(extracted_quotes, savepath)

        return extracted_quotes

    @staticmethod
    def get_quote(table) -> List[str]:
        """
        Extracts quote text from a given HTML table.

        Args:
            table (BeautifulSoup element): The table containing quotes.

        Returns:
            List[str]: A list of extracted quotes.
        """
        quotes = []
        try:
            for quote in table.find_all('div', style='font-size:12px;font-family:Arial;'):
                quotes.append(quote.get_text(strip=True))
        except AttributeError:
            return [""]
        return quotes

    @staticmethod
    def get_author(table) -> List[str]:
        """
        Extracts author names from a given HTML table.

        Args:
            table (BeautifulSoup element): The table containing author names.

        Returns:
            List[str]: A list of extracted author names.
        """
        authors = []
        try:
            for author in table.find_all('div', style='padding-top:2px;'):
                author_name = author.find('a').get_text(strip=True)
                authors.append(author_name)
        except AttributeError:
            return [""]
        return authors

    @staticmethod
    def get_tags(table) -> str:
        """
        Extracts the topic tag from a given HTML table.

        Args:
            table (BeautifulSoup element): The table containing tag information.

        Returns:
            str: The extracted topic tag.
        """
        try:
            tag = table.find('div', style='padding-top:10px;font-size:19px;font-family:Times New Roman;color:#347070;').get_text(strip=True)
        except AttributeError:
            return ""
        idx = tag.index(' Quote')
        return tag[:idx]

    @staticmethod
    def get_likes(table) -> None:
        """
        Extracts the number of likes for a quote (not available in this scraper).

        Args:
            table (BeautifulSoup element): The table containing like information.

        Returns:
            None: Since likes are not available on this website.
        """
        return None

    def scrape_topics(self, save: bool = False, savepath: str = 'topics_fq.pkl') -> List[Dict[str, str]]:
        """
        Scrapes all available topics and their URLs from the website.

        Args:
            save (bool): If True, saves the extracted topics to a pickle file.
            savepath (str): The path to save the topic data. Must be a .pkl or .json file

        Returns:
            List[Dict[str, str]]: A list of dictionaries containing topic names and URLs.
        """
        url = 'http://www.famousquotesandauthors.com/quotes_by_topic.html'
        base_quote_url = 'http://www.famousquotesandauthors.com'
        response = self.get_page(url)
        soup = self.parse_page(response)

        topics = []
        for topic_block in soup.find_all('tr', height='14'):
            topic = topic_block.find('a').get_text(strip=True)
            topic_url = urljoin(base_quote_url, topic_block.find('a').get('href'))
            topics.append({'topic': topic, 'url': topic_url})

        if save:
            self._save(topics, savepath)

        return topics

    def scrape_authors(self, save: bool = False, savepath: str = 'authors_fq.pkl') -> List[Dict[str, str]]:
        """
        Scrapes all authors and their respective URLs from the website.

        Args:
            save (bool): If True, saves the extracted author data to a pickle file.
            savepath (str): The path to save the author data. Must be a .pkl or .json file

        Returns:
            List[Dict[str, str]]: A list of dictionaries containing author names and URLs.
        """
        url = 'http://www.famousquotesandauthors.com/quotes_by_author.html'
        base_author_url = 'http://www.famousquotesandauthors.com'
        response = self.get_page(url)
        soup = self.parse_page(response)

        authors = []
        for author_block in soup.find_all('tr', height='15'):
            author = author_block.find('a').get_text(strip=True)
            author_url = urljoin(base_author_url, author_block.find('a').get('href'))
            authors.append({'author': author, 'url': author_url})

        if save:
            self._save(authors, savepath)

        return authors

    def scrape_many_pages(self, urls: List[str], save: bool = False, savepath: str = 'quotes_fq.pkl') -> List[Dict[str, str]]:
        """
        Scrapes multiple pages concurrently and extracts quotes.

        Args:
            urls (List[str]): A list of URLs to scrape.
            save (bool): If True, saves the extracted data to a pickle file.
            savepath (str): The path to save the extracted data. Must be a .pkl or .json file.

        Returns:
            List[Dict[str, str]]: A list of dictionaries containing quote data from all pages.
        """
        quotes = []
        futures = []

        with ThreadPoolExecutor(max_workers=10) as executor:
            for url in urls:
                future = executor.submit(self.scrape_page, url)
                futures.append(future)

            for future in as_completed(futures):
                quotes.extend(future.result())

        if save:
            self._save(quotes, savepath)

        return quotes
