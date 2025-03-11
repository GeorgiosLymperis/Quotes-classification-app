from base_scraper import BaseScraper
from string import ascii_lowercase
from urllib.parse import urljoin
from concurrent.futures import ThreadPoolExecutor, as_completed

class AzQuoteScraper(BaseScraper):
    def scrape_page(self, url, save: bool = False):
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
        return block.find('a', class_='title').get_text()
    
    @staticmethod
    def get_author(block):
        author_tag = block.find('div', class_='author').get_text(strip=True)
        return author_tag if author_tag else 'Unknown'
    
    @staticmethod
    def get_tags(block):
        tags = []
        tags_block = block.find('div', class_='mytags')
        for tag in tags_block.find_all('a'):
            tags.append(tag.get_text())

        return tags
    @staticmethod
    def get_likes(block):
        icons = block.find('div', class_='share-icons')
        return icons.find('a', class_='heart24 heart24-off').get_text()
    
    def scrape_topics_by_letter(self, letter, save=False):
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
                topics.append((topic_of_quote, urljoin(quote_url, topic_url)))

        if save == True:
            self._save(topics, 'topics_az.pkl')

        return topics
    
    def scrape_topics(self, save=False):
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
    
    def scrape_authors_page(self, letter, number, save=False):
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
            author_url = urljoin(quote_url, author_url)
            name, profession, birthday = [column.text.strip() for column in columns]
            authors_of_page.append((name, author_url, profession, birthday))

        if save == True:
            self._save(authors_of_page, 'authors.pkl')
        return authors_of_page
    
    def scrape_authors(self, save=False):
        authors = []
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = []
            for letter in ascii_lowercase:
                author_letter_url = f'https://www.azquotes.com/quotes/authors/{letter}/'
                response = self.get_page(author_letter_url + '1')
                soup = self.parse_page(response)
                num_of_pages = soup.find('div', class_='pager').find('span').get_text().split(' ')[-1]

                for i in range(1, int(num_of_pages) + 1):
                    future = executor.submit(self.scrape_authors_page, letter, i)
                    futures.append(future)

            for future in as_completed(futures):
                authors.extend(future.result())

        if save == True:
            self._save(authors, 'authors.pkl')
        return authors
    
    def scrape_many_pages(self, url, start_page, end_page, save=False):
        quotes = []
        with ThreadPoolExecutor(max_workers=10) as executor:
            futures = []
            for page in self.__generate_pages(url, start_page, end_page):
                future = executor.submit(self.scrape_page, page)
                futures.append(future)

            for future in as_completed(futures):
                quotes.extend(future.result())

        if save == True:
            self._save(quotes, 'quotes.pkl')
        return quotes
            
    def __generate_pages(self, url, start_page, end_page):
        base_url = urljoin(url, '?p=')
        for i in range(start_page, end_page + 1):
            yield urljoin(base_url, str(i))


if __name__ == '__main__':
    url = 'https://www.azquotes.com/quotes/topics/inspirational.html'
    scraper = AzQuoteScraper()
    quotes = scraper.scrape_page(url)
    print(quotes)