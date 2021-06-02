package org.yinzl.ElasticsearchPlugin.rescore;

import static java.util.Collections.singletonList;

import java.util.List;

import org.elasticsearch.plugins.Plugin;
import org.elasticsearch.plugins.SearchPlugin;

public class ExampleRescorePlugin extends Plugin implements SearchPlugin {
	@Override
	public List<RescorerSpec<?>> getRescorers() {
		return singletonList(new RescorerSpec<>(ExampleRescoreBuilder.NAME, ExampleRescoreBuilder::new,
				ExampleRescoreBuilder::fromXContent));
	}
}