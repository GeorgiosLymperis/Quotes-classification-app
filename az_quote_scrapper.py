from base_scraper import BaseScraper
from string import ascii_lowercase
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List

class AzQuoteScraper(BaseScraper):
    """
    A web scraper for AZQuotes that extracts quotes, topics, and authors from the site.
    
    This scraper supports:
    - Scraping individual pages for quotes
    - Extracting topics by letter
    - Scraping author details with multi-threading
    - Handling pagination for multiple pages
    - Saving extracted data as JSON or pickle
    """
    def scrape_page(self, url: str, save: bool = False):
        """
        Scrapes a single page for quotes and extracts relevant information.

        Args:
            url (str): The URL of the page to scrape.
            save (bool): If True, saves the data to a pickle file.

        Returns:
            list: A list of dictionaries containing quote data.
        """
        page = self.get_page(url)
        soup = self.parse_page(page)
        quotes = []
        for block in soup.find_all('div', class_='wrap-block'):
            quotes.append({
                'quote': self.get_quote(block),
                'author': self.get_author(block),
                'tags': self.get_tags(block),
                'likes': self.get_likes(block)
            })

        if save == True:
            self._save(quotes, 'quotes_az.pkl')        
        return quotes
    
    @staticmethod
    def get_quote(block):
        """
        Extracts the quote text from a quote block.

        Args:
            block (BeautifulSoup element): The block containing the quote.

        Returns:
            str: The extracted quote text.
        """
        return block.find('a', class_='title').get_text()
    
    @staticmethod
    def get_author(block):
        """
        Extracts the author of the quote.

        Args:
            block (BeautifulSoup element): The block containing the author.

        Returns:
            str: The extracted author name, or 'Unknown' if not found.
        """
        author_tag = block.find('div', class_='author').get_text(strip=True)
        return author_tag if author_tag else 'Unknown'
    
    @staticmethod
    def get_tags(block):
        """
        Extracts the tags associated with a quote.

        Args:
            block (BeautifulSoup element): The block containing tags.

        Returns:
            list: A list of tag strings.
        """
        tags = []
        tags_block = block.find('div', class_='mytags')
        for tag in tags_block.find_all('a'):
            tags.append(tag.get_text())

        return tags
    @staticmethod
    def get_likes(block):
        """
        Extracts the number of likes for a quote.

        Args:
            block (BeautifulSoup element): The block containing likes.

        Returns:
            str: The number of likes as a string.
        """
        icons = block.find('div', class_='share-icons')
        return icons.find('a', class_='heart24 heart24-off').get_text()
    
    def scrape_topics_by_letter(self, letter: str, save: bool = False):
        """
        Scrapes topics that start with a specific letter.

        Args:
            letter (str): The letter for which topics should be scraped.
            save (bool): If True, saves the data to a pickle file.

        Returns:
            list: A list of dictionaries containing topic names and URLs.
        """
        url = 'https://www.azquotes.com/quotes/tags/{}/'
        quote_url = 'https://www.azquotes.com'


        topics = []
        topic_url = url.format(letter)
        response = self.get_page(topic_url)
        soup = self.parse_page(response)
        soup = soup.find('section', class_='authors-page')
        for topic in soup.find_all('li'):
            topic_of_quote = topic.find('a').get_text(strip=True)
            topic_url = topic.find('a').get('href')
            if topic_of_quote:
                topics.append({'topic': topic_of_quote,'url': quote_url + topic_url})

        if save == True:
            self._save(topics, 'topics_az.pkl')

        return topics
    
    def scrape_topics(self, save: bool=False):
        """
        Scrapes all available topics from AZQuotes.

        Args:
            save (bool): If True, saves the data to a pickle file.

        Returns:
            list: A list of topic dictionaries.
        """
        topics = []
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = []
            for letter in ascii_lowercase:
                future = executor.submit(self.scrape_topics_by_letter, letter)
                futures.append(future)

            for future in as_completed(futures):
                topics.extend(future.result())

        if save == True:
            self._save(topics, 'topics_az.pkl')

        return topics
    
    def scrape_authors_page(self, letter: str, number: int, save=False):
        """
        Scrapes a specific page of authors for a given letter.

        Args:
            letter (str): The letter for author names.
            number (int): The page number to scrape.
            save (bool): If True, saves the data to a pickle file.

        Returns:
            list: A list of dictionaries containing author data.
        """
        url = 'https://www.azquotes.com/quotes/authors/{}/'
        quote_url = 'https://www.azquotes.com'

        authors_of_page = []
        author_letter_url = url.format(letter)
        response = self.get_page(author_letter_url + str(number))
        soup = self.parse_page(response)
        table_body = soup.find('tbody')

        for row in table_body.find_all('tr'):
            columns = row.find_all('td')
            author_url = row.find('a').get('href')
            author_url = quote_url + author_url
            name, profession, birthday = [column.text.strip() for column in columns]
            authors_of_page.append({'name': name,'url': author_url,'profession': profession,'birthday': birthday})

        if save == True:
            self._save(authors_of_page, 'authors.pkl')
        return authors_of_page
    
    def __find_num_of_pages(self, url: str) -> int:
        """
        Determines the number of available pages for a given URL.

        Args:
            url (str): The URL to check.

        Returns:
            int: The number of pages available.
        """
        response = self.get_page(url)
        soup = self.parse_page(response)
        pager_div = soup.find('div', class_='pager')
        if not pager_div:
            return 1
        page_spans = pager_div.find('span')
        try:
            num_of_pages = int(page_spans.get_text().split(' ')[-1])
        except ValueError:
            print(f"Could not extract number of pages for page {url}")
            num_of_pages = 1

        return num_of_pages
    
    def scrape_authors(self, save: bool=False):
        """
        Scrapes all authors listed on AZQuotes.

        Args:
            save (bool): If True, saves the data to a pickle file.

        Returns:
            list: A list of author dictionaries.
        """
        authors = []
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = []
            for letter in ascii_lowercase:
                author_letter_url = f'https://www.azquotes.com/quotes/authors/{letter}/'
                num_of_pages = self.__find_num_of_pages(author_letter_url + '1')

                for i in range(1, int(num_of_pages) + 1):
                    future = executor.submit(self.scrape_authors_page, letter, i)
                    futures.append(future)

            for future in as_completed(futures):
                authors.extend(future.result())

        if save == True:
            self._save(authors, 'authors_az.pkl')
        return authors
    
    def scrape_many_pages(self, urls:List[str], start_page: int, end_page: int, save: bool=False):
        """
        Scrapes multiple paginated URLs concurrently and extracts quotes.

        This method handles pagination dynamically, submitting scraping tasks 
        for each page across multiple URLs using threading.

        Args:
            urls (list): A list of base URLs to scrape.
            start_page (int): The starting page number for pagination.
            end_page (int): The last page number to scrape.
            save (bool): If True, saves the extracted data to a pickle file.

        Returns:
            list: A list of dictionaries containing quote data from all pages.
        """
        quotes = []
        futures = []
        len_urls = len(urls)
        with ThreadPoolExecutor(max_workers=10) as executor:
            for i, url in enumerate(urls):
                print(i, '/', len_urls)
                for page in self.__generate_pages(url, start_page, end_page):
                    future = executor.submit(self.scrape_page, page)
                    futures.append(future)

            for future in as_completed(futures):
                quotes.extend(future.result())

        if save == True:
            self._save(quotes, 'quotes.pkl')
        return quotes
            
    def __generate_pages(self, url: str, start_page: int, end_page: int):
        """
        Generates a list of paginated URLs for a given base URL.

        This method ensures that the generated page range does not exceed the 
        available number of pages on the website.

        Args:
            url (str): The base URL for pagination.
            start_page (int): The starting page number.
            end_page (int): The last page number to scrape.

        Returns:
            list: A list of fully constructed paginated URLs.
        """
        num_of_pages = self.__find_num_of_pages(url)
        end_page = num_of_pages if end_page > num_of_pages else end_page
        base_url = f'{url}?p='
        for i in range(start_page, end_page + 1):
            yield base_url + str(i)
